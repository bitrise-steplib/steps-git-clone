# Git clone step

Clones a specified git repository to the desired local path.

Can be run directly with the [bitrise CLI](https://github.com/bitrise-io/bitrise),
just `git clone` this repository, `cd` into it's folder in your Terminal/Command Line
and call `bitrise run test`.

*Check the `bitrise.yml` file for required inputs which have to be
added to your `.bitrise.secrets.yml` file!*


## Requirements

* `Go`: at least `1.5.x` (because of Go vendoring support)
