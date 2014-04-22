require 'base64'
require 'fileutils'
require 'uri'
require 'optparse'

options = {
  user_home: ENV['HOME']
}

opt_parser = OptionParser.new do |opt|
  opt.banner = "Usage: git_clone.rb [OPTIONS]"
  opt.separator  ""
  opt.separator  "Options (options without [] are required)"

  opt.on("--repo-url URL", "repository url") do |value|
    options[:repo_url] = value
  end

  opt.on("--branch [BRANCH]", "branch name") do |value|
    options[:branch] = value
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

unless options[:repo_url] and options[:repo_url].length > 0
  puts opt_parser
  exit
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


def add_host_to_known_hosts_if_needed(repo_url)
  #
  # regex based because it's used for repos with url like this: git@bitbucket.org:...-team/...-writer.git
  #  Ruby uri module doesn't support urls like this
  #

  host_regex = /@[a-z]*\.[a-z]*/
  match_res = host_regex.match(repo_url)
  unless match_res
    p "[!] Host can't be determined from url: #{repo_url}"
    return false
  end
  host = match_res[0][1..-1] # remove the @ from the beginning of the host url

  system "ssh -o StrictHostKeyChecking=no -o BatchMode=yes #{host}"
  return true
end



# -----------------------
# --- main
# -----------------------

#
prepared_repository_url = options[:repo_url]

begin
  URI.parse(prepared_repository_url)
rescue
  # probably an git ssh url like git@bitbucket.org:...
  # suppress known-host prompt
  add_host_to_known_hosts_if_needed(prepared_repository_url)
end

if options[:auth_ssh_key_base64] and options[:auth_ssh_key_base64].length > 0
  write_private_key_to_file(options[:user_home], options[:auth_ssh_key_base64])

  # suppress known-host prompt
  add_host_to_known_hosts_if_needed(prepared_repository_url)
elsif options[:auth_username] and options[:auth_username].length > 0 and options[:auth_password] and options[:auth_password].length > 0
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
if options[:branch] and options[:branch].length > 0
  git_branch_parameter = "--single-branch --branch #{options[:branch]}"
else
  git_branch_parameter = "--no-single-branch"
end

# GIT_ASKPASS=echo : this will automatically fail if git would show a password prompt
full_git_clone_command_string = "git clone --recursive #{git_branch_parameter} #{prepared_repository_url} #{options[:clone_destination_dir]}"

system "export GIT_ASKPASS=echo"
system 'export GIT_SSH="$PWD/ssh_no_prompt.sh"'
puts "$ #{full_git_clone_command_string}"
system full_git_clone_command_string


