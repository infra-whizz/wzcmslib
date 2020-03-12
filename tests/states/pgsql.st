id: pgsql
description: PostgreSQL management
state:
  update-pgsql:
    - packaging.os.apt:
        updated: pgsql

  install-pgsql ?is_pgsql_needed:
    - packaging.os.apt:
        present: pgsql
