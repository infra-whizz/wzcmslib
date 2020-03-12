#
# Domain Language Design
#
# "~"  -- reference to to include other state/job
# "&"  -- reference of job dependency (should be performed before)
# "?"  -- function returns a condition (Ansible's "when")
# "[]" -- function returns a list of objects and applies N times to the operand

id: state-definition
description: This is the original state, which defines state functions and syntax.
state:                                       # State tree
  an-example:                                # An ID of some job
    - shell:                                 # Call a shell command (no plugins)
        - get-kernel-info: "uname -a"        # Call an actual shell command
        - another-command: "cat /etc/hosts"  # Perform some other command

    - system.service:                        # Call ansible module: "ansible.<namespace>.<module>"
        name: httpd                          # Key/value parameter
        state: stopped                       # Key/value parameter

  install-postgres &pgsql/install-pgsql:     # Include "install-pgsql" job before PgSQL start.
    - system.service:                        # Note, that "install-pgsql" will resolve on its own.
        name: postgresql                     # See "pgsql.st" how it is done.
        state: started

  # Inclusion is a path of the following format:
  #
  #    ~<STATE-ID>/[BLOCK]
  #
  # Examples:
  #
  # Include only one block:
  #    ~my-state-id/installs-something
  #
  # Include few blocks:
  #    ~my-state-id/install-postgres:add-users
  #
  # Include everything:
  #    ~my-state-id

  #~pgsql-id/update-pgsql ?pgsql_present:
  #~pgsql-id ?pgsql_present:

  install-emacs-apt ?is_debian_family:
    - packaging.os.apt:
        present: emacs-nox

  install-emacs-yum ?is_redhat_family:
    - packaging.os.yum:
        present: emacs-nox

  add-josh:
    - sysem.user:
        name: josh
        state: present

  # Iteration. Function "more_users" is expected to return an array of
  # keywords, like [{'key': 'val'}, {'key': 'val'}, {..}...]
  # For example, instead of writing 20 users in this state, return them from
  # the function "more_users" (see 'definition.fc')
  add-some-more-users:
    - system.user []more_users:  # Iterate over an array of keyword parameters (dicts).
