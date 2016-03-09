## Changelog (Current version: 3.1.1)

-----------------

### 3.1.1 (2016 Mar 09)

* [ff7d9ea] release configs
* [532ad02] Merge pull request #10 from bitrise-io/log_checkout_commit_hash
* [a0a399c] print commit hash, even if empty
* [cb73b60] log git clone hash
* [e1d080b] share-this-step workflow

### 3.1.0 (2015 Nov 06)

* [bae5898] style revision
* [d101329] further unused code/option cleanup
* [8016714] removed unnecessary formatted output path log
* [fb36fec] removed unused parameters; does not log sensitive info (ssh key) anymore; README update; testing revision
* [864ca9d] Merge pull request #7 from bazscsa/patch-1
* [ed5879d] Update step.yml

### 3.0.0 (2015 Sep 11)

* [c44326a] removed the 'destination dir will be removed' note from step.yml
* [ba0118e] no more destination path magic, and DON'T DELETE IT!! - it's just plain wrong to do it without even asking!

### 2.2.0 (2015 Sep 11)

* [83cde19] Merge pull request #5 from gkiki90/master
* [98845eb] clone destination path is required
* [1a01db1] clone_into_dir fix

### 2.1.0 (2015 Sep 10)

* [23a96b4] bitrise.yml update
* [cf5bd15] Merge pull request #4 from gkiki90/script_dir
* [7a9a26f] fix

### 2.0.0 (2015 Sep 04)

* [ab46847] `step.yml` description fix
* [843b1d1] updated for Bitrise "V2" & stripped down (removed deprecated&unused) : advanced options moved into `steps-git-clone-extended`
* [5abbf32] converted to V2 step format, ready to run with `bitrise` CLI
* [f365185] indent fix

### 1.5.0 (2015 Apr 11)

* [fa0f15f] don't fail if there's no checkout parameter (for compatibility reasons) but print a debug message
* [28ee85e] fail if no checkout parameter specified
* [dc8fef1] git_clone converted to tabs
* [b6c73b8] removed empty line
* [dc701c8] deprecated comment moved
* [accb021] removed base64 ssh key again (got back during the revision)
* [17f8e43] step.yml revision
* [81f82e6] minimal step.yml syntax fix&revision
* [d1bfe5d] deprecated in git_clone too
* [6603648] marked base64 key as deprecated
* [00795ff] removed old base64 ssh key input
* [5c1d98a] whitespace and indent fix
* [97b2c37] Merge pull request #2 from birmacher/master
* [f651d16] GitHub pull request support
* [7f02dd1] fix to clone repository without master branch

### 1.4.0 (2014 Nov 12)

* [339b471] step sh style fix
* [13a1b68] step.yml revision; exported outputs support (git commit hash, msg, author, ...); a bit of formatted output handling revision
* [db21a9e] Merge pull request #1 from erosdome/master
* [d33e6d2] Update step.yml
* [210deb8] Update step.yml
* [d17fd22] Update README.md

### 1.3.0 (2014 Oct 17)

* [0f93b77] Merge branch 'release/1.3.0'
* [d368f9b] comment/syntax fix
* [c04c9ca] the multiline ssh-key parameter is now retrieved directly from the environment
* [2563aa1] raw ssh key support

### 1.2.0 (2014 Jul 11)

* [22044d4] Merge branch 'release/1.2.0'
* [ee25b58] rename from codename concrete to Bitrise

### 1.1.3 (2014 Jun 24)

* [10a631f] Merge branch 'release/1.1.3'
* [58252bf] highlight the commit-hash in formatted output with pre/code

### 1.1.2 (2014 Jun 24)

* [625dd85] Merge branch 'release/1.1.2'
* [2ebbde6] commit hash formatted output formatting fix

### 1.1.1 (2014 Jun 24)

* [31d82bb] Merge branch 'release/1.1.1'
* [7c772d2] option to generate a formatted output (markdown) file

### 1.1.0 (2014 Jun 17)

* [567d24b] Merge branch 'release/1.1.0'
* [0c6c170] better private key handling: won't overwrite id_rsa but rather use a 'concrete' ssh private key file (and use it specifically!) + it will remove the private key file at the end of the script

### 1.0.9 (2014 Jun 17)

* [df8ebbf] Merge branch 'release/1.0.9'
* [5d5c536] don't redirect user-known-hosts-file to /dev/null

### 1.0.8 (2014 Jun 14)

* [fb305c3] Merge branch 'release/1.0.8'
* [03a74b1] 'this script path' handling change/fix

### 1.0.7 (2014 Jun 14)

* [1d10e27] Merge branch 'release/1.0.7'
* [b02e44f] path handling fix

### 1.0.6 (2014 Jun 14)

* [eda6553] Merge branch 'release/1.0.6'
* [7b8a61b] unused clone cleanup
* [431b802] commit-hash parameter support + a significant rewrite to remove old, now not required workarounds and to support clone+checkout based on commit-hash

### 1.0.5 (2014 Jun 11)

* [7ff11d1] Merge branch 'release/1.0.5'
* [1f77349] ssh no prompt: redirect known hosts file to /dev/null

### 1.0.4 (2014 Jun 04)

* [1250ced] Merge branch 'release/1.0.4'
* [1fd3028] clean up the clone-destination-dir before cloning

### 1.0.3 (2014 May 30)

* [1d8c023] Merge branch 'release/1.0.3'
* [f1048ec] retry delay increased

### 1.0.2 (2014 May 29)

* [98d9e19] Merge branch 'release/1.0.2'
* [9f053ed] retry with delay support

-----------------

Updated: 2016 Mar 09