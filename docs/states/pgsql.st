id: pgsql-id
description: PostgreSQL management
state:
  update-pgsql:
    - packaging.os.apt:
        - updated: pgsql

  install-pgsql:
    - packaging.os.apt:
        - present: pgsql
