import sys

from cement.utils import fs
from tinydb import TinyDB

from esper.ext.db_wrapper import DBWrapper


def extend_tinydb(app):
    db_file = app.config.get('esper', 'creds_file')
    app.log.debug(f"DB File path from config: {db_file}")

    # ensure that we expand the full path
    db_file = fs.abspath(db_file)
    app.log.debug(f"Absolute DB File path: {db_file}")

    # Create the parent folder(s)
    app.log.debug(f"Creating parent folders for DB File...")
    fs.ensure_parent_dir_exists(db_file)

    # Create and assign the DB file
    app.log.debug(f"Assigning DB object to app -> app.db")
    app.extend('creds', TinyDB(db_file))


def validate_creds_exists(app):
    db = DBWrapper(app.creds)
    if not db.get_configure():
        app.log.error("Credentials have not been set!")
        app.log.info("Setup credentials by calling `configure` command.")

        sys.exit(1)
