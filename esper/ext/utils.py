from cement.utils import fs
from tinydb import TinyDB
from esperclient.configuration import Configuration
import esperclient as client


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


def get_client_config(app):

    creds = app.creds.all()[0]

    config = Configuration()
    config.username = creds["username"]
    config.password = creds["password"]
    config.host = creds["host"]

    return config


def get_device_api_instance(config):
    return client.DeviceApi(client.ApiClient(config))
