require 'base64'
require 'fileutils'
require 'uri'
require 'optparse'

options = {
  user_home: ENV['HOME'],
  retry_count: 2, # if first attempt fails retry x times
  retry_delay_secs: 2 # delay between retrys
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

  opt.on("--dest-dir [DESTINATIONDIR]", "local clone destination directory path") do |value|
    options[:clone_destination_dir] = value
  end

  opt.on("--auth-username [USERNAME]", "username for authentication - requires --auth-password to be specified") do |value|
    options[:auth_username] = value
  end

  opt.on("--auth-password [PASSWORD]", "password for authentication - requires --auth-username to be specified") do |value|
    options[:auth_password] = value
  end

  opt.on("--auth-ssh-base64 [SSH-BASE64]", "Base64 representation of the ssh private key to be used") do |value|
    options[:auth_ssh_key_base64] = value
  end

  opt.on("-h","--help","Shows this help message") do
    puts opt_parser
  end
end

opt_parser.parse!

puts "Provided options: #{options}"

unless options[:repo_url] and options[:repo_url].length > 0
  puts opt_parser
  exit 1
end



# -----------------------
# --- functions
# -----------------------


def write_private_key_to_file(user_home, auth_ssh_private_key_base64)
  private_key_file_path = File.join(user_home, '.ssh/id_rsa')

  # create the folder if not yet created
  FileUtils::mkdir_p(File.dirname(private_key_file_path))

  # private key - save to file
  private_key_decoded = Base64.strict_decode64(auth_ssh_private_key_base64)
  File.open(private_key_file_path, 'wt') { |f| f.write(private_key_decoded) }
  system "chmod 600 #{private_key_file_path}"
end


# -----------------------
# --- main
# -----------------------

#
prepared_repository_url = options[:repo_url]

used_auth_type=nil
if options[:auth_ssh_key_base64] and options[:auth_ssh_key_base64].length > 0
  used_auth_type='ssh'
  write_private_key_to_file(options[:user_home], options[:auth_ssh_key_base64])
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
git_branch_parameter = ""
if options[:tag] and options[:tag].length > 0
  # since git 1.8.x tags can be specified as "branch" too ( http://git-scm.com/docs/git-clone )
  #  [!] this will create a detached head, won't switch to a branch!
  git_branch_parameter = "--single-branch --branch #{options[:tag]}"
elsif options[:branch] and options[:branch].length > 0
  git_branch_parameter = "--single-branch --branch #{options[:branch]}"
else
  git_branch_parameter = "--no-single-branch"
end

# GIT_ASKPASS=echo : this will automatically fail if git would show a password prompt
full_git_clone_command_string = "git clone --recursive --depth 1 #{git_branch_parameter} #{prepared_repository_url} #{options[:clone_destination_dir]}"

this_script_path = File.expand_path(File.dirname(File.dirname(__FILE__)))
puts "$ #{full_git_clone_command_string}"
full_cmd_string="GIT_ASKPASS=echo GIT_SSH=\"#{this_script_path}/ssh_no_prompt.sh\" #{full_git_clone_command_string}"
if used_auth_type=='ssh'
  system(%Q{expect -c "spawn ssh-add $HOME/.ssh/id_rsa; expect -r \"Enter\";"})
end
full_cmd_string = "ssh-agent bash -c '#{full_cmd_string}'"

$options = options
def do_clone_command(cmd_string, retry_count=0)
  is_clone_success=system(cmd_string)

  if not is_clone_success and retry_count < $options[:retry_count]
    sleep $options[:retry_delay_secs]
    retry_count = retry_count+1
    puts "Attempt failed - retry... (#{retry_count} / 2)"
    is_clone_success = do_clone_command(cmd_string, retry_count)
  end

  return is_clone_success
end

is_clone_success = do_clone_command(full_cmd_string)
puts "Clone Is Success?: #{is_clone_success}"

exit (is_clone_success ? 0 : 1)


