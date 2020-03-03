#
# Domain Language Design
#
# "~"  -- reference to to include other state/job
# "&"  -- reference of job dependency (should be performed before)
# "?"  -- function returns a condition (Ansible's "when")
# "[]" -- function returns a list of objects (Ansible's "loop") and applies N times to the operand
# "{}" -- function returns a dict and applies only once to the operand

id: some-id
description: description text comes here
state:                                    # State tree
  some-batch-id &pgsql-id/install-pgsql:  # ID of some job, require "install-pgsql" job from "pgsql-id" state.
    - shell:                              # Call a shell command (no plugins)
      - command-id: "uname -a"            # The actual command
      - another-command: "ls"             # Some other command

    - system.service:                     # Call ansible module: "ansible.<namespace>.<module>"
      - name: postgresql                  # Key/value parameter
      - state: stopped                    # Key/value parameter

    - system.service:
      - name: postgresql
      - state: stopped

  ~pgsql-id/update-pgsql ?pgsql_present:

  install-emacs-apt ?is_debian_family:
    - packaging.os.apt:
      - present: emacs-nox

  add-users:
    - system.user []my_users:

  add-one-user:
    - system.user {}my_user:
