require 'base64'
require 'fileutils'
require 'uri'
require 'optparse'

options = {
	user_home: ENV['HOME'],
	private_key_file_path: nil,
	formatted_output_file_path: nil,
	output_env_file_path: "#{ENV['HOME']}/.bash_profile",
	is_export_outputs: false
}

opt_parser = OptionParser.new do |opt|
	opt.banner = "Usage: git_clone.rb [OPTIONS]"
	opt.separator  ""
	opt.separator  "Options (options without [] are required)"

	opt.on("--repo-url URL", "repository url") do |value|
		options[:repo_url] = value
	end

	opt.on("--branch [BRANCH]", "branch name. IMPORTANT: if tag is specified the branch parameter will be ignored!") do |value|
		options[:branch] = value
	end

	opt.on("--tag [TAG]", "tag name. IMPORTANT: if tag is specified the branch parameter will be ignored!") do |value|
		options[:tag] = value
	end

	opt.on("--commit-hash [COMMITHASH]", "commit hash. IMPORTANT: if commit-hash is specified the branch and tag parameters will be ignored!") do |value|
		options[:commit_hash] = value
	end

	opt.on("--pull-request [PULL-REQUEST-ID]", "pull request id. IMPORTANT: works only with GitHub") do |value|
		options[:pull_request_id] = value
	end

	opt.on("--dest-dir [DESTINATIONDIR]", "local clone destination directory path") do |value|
		options[:clone_destination_dir] = value
	end

	opt.on("--auth-username [USERNAME]", "username for authentication - requires --auth-password to be specified") do |value|
		options[:auth_username] = value
	end

	opt.on("--auth-password [PASSWORD]", "password for authentication - requires --auth-username to be specified") do |value|
		options[:auth_password] = value
	end

	# DEPRECATED!
	opt.on("--auth-ssh-base64 [SSH-BASE64]", "Base64 representation of the ssh private key to be used") do |value|
		options[:auth_ssh_key_base64] = value
	end

	opt.on("--formatted-output-file [FILE-PATH]", "If given a formatted (markdown) output will be generated") do |value|
		options[:formatted_output_file_path] = value
	end

	opt.on("--is-export-outputs [true/false]", "if false (default) then it won't export it's outputs. If true then it will.") do |value|
		if value == 'true'
			options[:is_export_outputs] = true
		end
	end

	opt.on("-h","--help","Shows this help message") do
		puts opt_parser
	end
end

ssh_key_from_env = ENV['auth_ssh_private_key']
if ssh_key_from_env
	options[:auth_ssh_key_raw] = ssh_key_from_env
end

opt_parser.parse!

if options[:formatted_output_file_path] and options[:formatted_output_file_path].length < 1
	options[:formatted_output_file_path] = nil
end

puts "Provided options: #{options}"

unless options[:repo_url] and options[:repo_url].length > 0
	puts opt_parser
	exit 1
end



# -----------------------
# --- functions
# -----------------------


def write_private_key_to_file(user_home, auth_ssh_private_key)
	private_key_file_path = File.join(user_home, '.ssh/bitrise')

	# create the folder if not yet created
	FileUtils::mkdir_p(File.dirname(private_key_file_path))

	# private key - save to file
	File.open(private_key_file_path, 'wt') { |f| f.write(auth_ssh_private_key) }
	system "chmod 600 #{private_key_file_path}"

	return private_key_file_path
end


# -----------------------
# --- main
# -----------------------

# normalize input pathes
options[:clone_destination_dir] = File.expand_path(options[:clone_destination_dir])
if options[:formatted_output_file_path]
	options[:formatted_output_file_path] = File.expand_path(options[:formatted_output_file_path])
end


#
prepared_repository_url = options[:repo_url]

used_auth_type=nil
if options[:auth_ssh_key_raw] and options[:auth_ssh_key_raw].length > 0
	used_auth_type='ssh'
	options[:private_key_file_path] = write_private_key_to_file(options[:user_home], options[:auth_ssh_key_raw])
elsif options[:auth_ssh_key_base64] and options[:auth_ssh_key_base64].length > 0
	used_auth_type='ssh'
	private_key_decoded = Base64.strict_decode64(options[:auth_ssh_key_base64])
	options[:private_key_file_path] = write_private_key_to_file(options[:user_home], private_key_decoded)
elsif options[:auth_username] and options[:auth_username].length > 0 and options[:auth_password] and options[:auth_password].length > 0
	used_auth_type='login'
	repo_uri = URI.parse(prepared_repository_url)

	# set the userinfo
	repo_uri.userinfo = "#{options[:auth_username]}:#{options[:auth_password]}"
	# 'serialize'
	prepared_repository_url = repo_uri.to_s
else
	# Auth: No Authentication information found - trying without authentication
end

# do clone
git_checkout_parameter = nil
# git_branch_parameter = ""
if options[:pull_request_id] and options[:pull_request_id].length > 0
	git_checkout_parameter = "pull/#{options[:pull_request_id]}"
elsif options[:commit_hash] and options[:commit_hash].length > 0
	git_checkout_parameter = options[:commit_hash]
elsif options[:tag] and options[:tag].length > 0
	# since git 1.8.x tags can be specified as "branch" too ( http://git-scm.com/docs/git-clone )
	#  [!] this will create a detached head, won't switch to a branch!
	# git_branch_parameter = "--single-branch --branch #{options[:tag]}"
	git_checkout_parameter = options[:tag]
elsif options[:branch] and options[:branch].length > 0
	# git_branch_parameter = "--single-branch --branch #{options[:branch]}"
	git_checkout_parameter = options[:branch]
else
	# git_branch_parameter = "--no-single-branch"
	puts " [!] No checkout parameter found"
end



$options = options
$prepared_repository_url = prepared_repository_url
$git_checkout_parameter = git_checkout_parameter
$this_script_path = File.expand_path('.')

class String
	def prepend_lines_with(prepend_with_string)
		return self.gsub(/^.*$/, prepend_with_string.to_s+'\&')
	end
end

def write_string_to_formatted_output(str_to_write)
	formatted_output_file_path = $options[:formatted_output_file_path]
	if formatted_output_file_path
		File.open(formatted_output_file_path, "w+") { |f|
			f.puts(str_to_write)
		}
	end
end

def bash_export_escape(str)
	return str.gsub('"'){'\\"'}
end

def export_step_output(key, value)
	File.open($options[:output_env_file_path], "a") { |f|
		f.puts %Q{export #{key}="#{bash_export_escape(value)}"}
	}
end

def do_clone()
	# first delete the destination folder - for git, especially if it's a retry
	return false unless system(%Q{rm -rf "#{$options[:clone_destination_dir]}"})
	# (re-)create
	return false unless system(%Q{mkdir -p "#{$options[:clone_destination_dir]}"})

	is_clone_success = false
	Dir.chdir($options[:clone_destination_dir]) do
		begin
			unless system(%Q{git init})
				raise 'Could not init git repository'
			end

			ssh_no_prompt_file = 'ssh_no_prompt.sh'
			if $options[:private_key_file_path]
				ssh_no_prompt_file = 'ssh_no_prompt_with_id.sh'
			end

			unless system(%Q{GIT_ASKPASS=echo GIT_SSH="#{$this_script_path}/#{ssh_no_prompt_file}" git remote add origin "#{$prepared_repository_url}"})
				raise 'Could not add remote'
			end

			fetch_command = "git fetch"
			if $options[:pull_request_id] and $options[:pull_request_id].length > 0
				fetch_command += " origin pull/#{$options[:pull_request_id]}/merge:#{$git_checkout_parameter}"
			end
			unless system(%Q{GIT_ASKPASS=echo GIT_SSH="#{$this_script_path}/ssh_no_prompt.sh" #{fetch_command}})
				raise 'Could not fetch from repository'
			end

			if $git_checkout_parameter != nil
				unless system("git checkout #{$git_checkout_parameter}")
					raise "Could not do checkout #{$git_checkout_parameter}"
				end

				unless system(%Q{GIT_ASKPASS=echo GIT_SSH="#{$this_script_path}/ssh_no_prompt.sh" git submodule update --init --recursive})
					raise 'Could not fetch from submodule repositories!'
				end

				# git clone stats
				commit_hash_str = `git log -1 --format="%H"`.chomp
				commit_msg_subject_str = `git log -1 --format="%s"`.chomp
				commit_msg_body_str = `git log -1 --format="%b"`.chomp
				commit_author_name_str = `git log -1 --format="%an"`.chomp
				commit_author_email_str = `git log -1 --format="%ae"`.chomp
				commit_commiter_name_str = `git log -1 --format="%cn"`.chomp
				commit_commiter_email_str = `git log -1 --format="%ce"`.chomp


				if $options[:is_export_outputs]
					export_step_output('GIT_CLONE_COMMIT_HASH', commit_hash_str)
					export_step_output('GIT_CLONE_COMMIT_MESSAGE_SUBJECT', commit_msg_subject_str)
					export_step_output('GIT_CLONE_COMMIT_MESSAGE_BODY', commit_msg_body_str)
					export_step_output('GIT_CLONE_COMMIT_AUTHOR_NAME', commit_author_name_str)
					export_step_output('GIT_CLONE_COMMIT_AUTHOR_EMAIL', commit_author_email_str)
					export_step_output('GIT_CLONE_COMMIT_COMMITER_NAME', commit_commiter_name_str)
					export_step_output('GIT_CLONE_COMMIT_COMMITER_EMAIL', commit_commiter_email_str)
				end


				formatted_output_file_path = $options[:formatted_output_file_path]
				if formatted_output_file_path
					commit_log_str = `git log -n 1 --tags --branches --remotes --format="fuller"`
					commit_log_str = commit_log_str.prepend_lines_with('    ')
					write_string_to_formatted_output(%Q{
# Commit Hash

	#{commit_hash_str}

# Commit Log

	#{commit_log_str}
})
				end
			else
				puts " (!) No checkout parameter (branch, tag, commit hash or pull-request ID) provided!"
			end

			is_clone_success = true
		rescue => ex
			puts "Error: #{ex}"
		end
	end

	unless is_clone_success
		# delete it
		system(%Q{rm -rf "#{$options[:clone_destination_dir]}"})
	end

	return is_clone_success
end

is_clone_success = do_clone()
puts "Clone Is Success?: #{is_clone_success}"

if options[:private_key_file_path]
	puts " (i) Removing private key file: #{options[:private_key_file_path]}"
	system(%Q{rm -P #{options[:private_key_file_path]}})
end

exit (is_clone_success ? 0 : 1)
