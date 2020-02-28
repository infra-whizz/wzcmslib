#
# Domain Language Design
#
# "&"  -- reference to other state/id
# "?"  -- function returns a condition (Ansible's "when")
# "[]" -- function returns a list of objects (Ansible's "loop") and applies N times to the operand
# "{}" -- function returns a dict and applies only once to the operand

id: some-id
description: description text comes here
include:  # Include state with pgsql batches/jobs
  - pgsql

state:                                # State tree
  some-batch-id:                      # ID of some job
    - shell:                          # Call a shell command (no plugins)
      - command-id: "uname -a"        # The actual command
      - another-command: "ls"         # Some other command

    - system.service:                 # Call ansible module: "ansible.<namespace>.<module>"
      - name: postgresql              # Key/value parameter
      - state: stopped                # Key/value parameter

    - system.service &install-pgsql:  # Require on "install-pgsql" state
      - name: postgresql
      - state: stopped

  install-emacs-apt ?is_debian_family:
    - packaging.os.apt:
      - present: emacs-nox

  add-users:
    - system.user []my_users:

  add-one-user:
    - system.user {}my_user:
