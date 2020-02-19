# ansibleremote

This is a local caller for Ansible Python modules (for now). Todo: call binary modules.
The reason is to make a static binary that would be able to run any type of Ansible modules,
yet not depend on an interpreter. That said, if a target machine has no Python interpreter
installed but has "binary" type of Ansible modules available, this is when `ansiblerunner`
comes to the rescue.

Assuming Ansible is installed on the system, this is an example call:

	ansiblerunner commands.command argv='uname -a'

The `ansiblerunner` is not supposed to be used directly. Just use Ansible instead. :-)
