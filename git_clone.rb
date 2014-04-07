require 'base64'
require 'fileutils'
require 'uri'

options = {
  GIT_BRANCH: ENV['GIT_BRANCH'],
  GIT_REPOSITORY_URL: ENV['GIT_REPOSITORY_URL'],
  CLONE_DESTINATION_DIR: ENV['CLONE_DESTINATION_DIR'],
  #
  USER_HOME: ENV['HOME'],
  # auth
  AUTH_USER: ENV['AUTH_USER'],
  AUTH_PASSWORD: ENV['AUTH_PASSWORD'],
  AUTH_SSH_PRIVATE_KEY_BASE64: ENV['AUTH_SSH_PRIVATE_KEY_BASE64']
}

p "options: #{options}"


def write_private_key_to_file(user_home, auth_ssh_private_key_base64)
  private_key_file_path = File.join(user_home, '.ssh/id_rsa')
  p "private_key_file_path: #{private_key_file_path}"
  # create the folder if not yet created
  FileUtils::mkdir_p(File.join(user_home, '.ssh'))
  # private key - save to file
  private_key_decoded = Base64.decode64(auth_ssh_private_key_base64)
  File.open(private_key_file_path, 'wt') { |f| f.write(private_key_decoded) }
  system "chmod 600 #{private_key_file_path}"
  p "Private key written to file."
end

def add_host_to_known_hosts_if_needed(repo_url)
  host_regex = /@[a-z]*\.[a-z]*/
  match_res = host_regex.match(repo_url)
  unless match_res
    p "[!] Host can't be determined from url: #{repo_url}"
    return false
  end
  host = match_res[0][1..-1] # remove the @ from the beginning of the host url
  p "Host url: #{host}"
  system "ssh -o StrictHostKeyChecking=no #{host}"
  return true
end

prepared_repository_url = options[:GIT_REPOSITORY_URL]

if options[:AUTH_SSH_PRIVATE_KEY_BASE64] and options[:AUTH_SSH_PRIVATE_KEY_BASE64].length > 0
  p "[i] Auth: using SSH private key (base64)"
  write_private_key_to_file(options[:USER_HOME], options[:AUTH_SSH_PRIVATE_KEY_BASE64])
  add_host_to_known_hosts_if_needed(prepared_repository_url)
elsif options[:AUTH_USER] and options[:AUTH_USER].length > 0 and options[:AUTH_PASSWORD] and options[:AUTH_PASSWORD].length > 0
  p "[i] Auth: with username and password"
  # https://viktorbenei@bitbucket.org/concrete-team/step-environment-writer.git
  repo_prefix_regex = /^https?:\/\/[a-z]*@/
  rres = repo_prefix_regex.match(prepared_repository_url)
  unless rres
    p "[!] Invalid url prefix: should start with 'http(s)://...@'"
  else
    http_part = /^https?:\/\//.match(prepared_repository_url)[0]
    # strip out the "prefix" part
    prepared_repository_url = prepared_repository_url[rres[0].length .. -1]
    # and recunstruct, with the auth parameters
    prepared_repository_url = "#{http_part}#{options[:AUTH_USER]}:#{options[:AUTH_PASSWORD]}@#{prepared_repository_url}"
  end
else
  p "[i] Auth: No Authentication information found - trying without authentication"
end

# do clone
p "prepared_repository_url: #{prepared_repository_url}"

git_branch_parameter = ""
if options[:GIT_BRANCH] and options[:GIT_BRANCH].length > 0
  git_branch_parameter = "--single-branch --branch #{options[:GIT_BRANCH]}"
else
  git_branch_parameter = "--no-single-branch"
end

full_git_clone_command_string = "git clone --recursive #{git_branch_parameter} #{prepared_repository_url} #{options[:CLONE_DESTINATION_DIR]}"
p "full_git_clone_command_string: #{full_git_clone_command_string}"
# system full_git_clone_command_string


