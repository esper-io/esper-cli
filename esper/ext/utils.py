import sys

import esperclient as client
from cement.utils import fs
from esperclient.configuration import Configuration
from tinydb import TinyDB


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
    if len(app.creds.all()) == 0:
        app.log.error("Credentials have not been set!")
        app.log.info("Setup credentials by calling `configure` command.")

        sys.exit(1)


def get_client_config(app):
    creds = app.creds.all()[0]

    config = Configuration()
    config.username = creds["username"]
    config.password = creds["password"]
    config.host = get_api_endpoint(creds["host"])

    return config


def get_device_api_instance(config):
    return client.DeviceApi(client.ApiClient(config))


def get_application_api_instance(config):
    return client.ApplicationApi(client.ApiClient(config))


def get_application_upload_api_instance(config):
    return client.EnterpriseApplicationApi(client.ApiClient(config))


def get_command_api(config):
    return client.CommandsApi(client.ApiClient(config))


def get_api_endpoint(host):
    return f'https://{host}-api.shoonyacloud.com/api'


def get_command_request(command_name, command_args):
    return client.CommandRequest(command_args=command_args,
                                 command=command_name)  # CommandRequest | command name to fire
