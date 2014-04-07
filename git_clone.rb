require 'base64'
require 'fileutils'

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

if options[:AUTH_SSH_PRIVATE_KEY_BASE64]
  private_key_file_path = File.join(options[:USER_HOME], '.ssh/id_rsa')
  p "private_key_file_path: #{private_key_file_path}"
  # create the folder if not yet created
  FileUtils::mkdir_p(File.join(options[:USER_HOME], '.ssh'))
  # private key - save to file
  private_key_decoded = Base64.decode64(options[:AUTH_SSH_PRIVATE_KEY_BASE64])
  File.open(private_key_file_path, 'wt') { |f| f.write(private_key_decoded) }
  system "chmod 600 #{private_key_file_path}"
end
