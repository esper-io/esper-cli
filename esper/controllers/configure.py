import json
from http import HTTPStatus

from cement import Controller, ex
from clint.textui import prompt
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists


class Configure(Controller):
    class Meta:
        label = 'configure'
        stacked_type = 'embedded'
        stacked_on = 'base'

        # text displayed at the top of --help output
        description = 'Esper CLI tool to manage resources on Esper.io API service'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli configure'

    @ex(
        help='Configure the credentials for `esper.io` API Service',
        arguments=[
            (['-s', '--set'],
             {'help': 'Create or Update credentials for Esper.io API Service',
              'action': 'store_true',
              'default': False,
              'dest': 'set'}),

            (['-l', '--list'],
             {'help': 'List credentials for Esper.io API Service',
              'action': 'store_true',
              'default': False,
              'dest': 'list'}),

            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ],
    )
    def configure(self):
        """Configure the credentials and host endpoints of the customer"""

        # Trigger the Insert operation, if --set is given OR if the Creds DB is empty
        db = DBWrapper(self.app.creds)
        credentials = db.get_configure()

        if self.app.pargs.set or not credentials:
            environment = input("Environment name: ")
            api_key = prompt.query("Esper API Key: ")

            token_client = APIClient({'api_key': api_key, 'environment': environment}).get_token_api_client()
            try:
                response = token_client.get_token_info()
            except ApiException as e:
                self.app.log.error(f"[configure] Failed to get token info: {e}")
                if e.status == HTTPStatus.UNAUTHORIZED:
                    self.app.render("You are not authorized, invalid API Key.\n")
                else:
                    error_message = json.loads(e.body).get('message') if e.body and json.loads(e.body).get(
                        'message') else e.reason
                    self.app.render(f"ERROR: {error_message}\n")
                return

            if response:
                enterprise_id = response.enterprise
            else:
                self.app.log.info(f"[configure] API key is not associated with any enterprise.")
                self.app.render("API key is not associated with any enterprise.\n")
                return

            credentials = {
                "environment": environment,
                "api_key": api_key,
                "enterprise_id": enterprise_id
            }

            # set new credentials into the DB
            self.app.log.debug("Purging and inserting new credentials DB...")
            db.set_configure(credentials)

        # Trigger listing operation, if --list is given or Creds DB has content
        if self.app.pargs.list or credentials:
            validate_creds_exists(self.app)

            if not self.app.pargs.json:
                title = "TITLE"
                details = "DETAILS"
                renderable = [
                    {title: 'environment', details: credentials.get('environment')},
                    {title: 'api_key', details: credentials.get('api_key')}
                ]
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys",
                                tablefmt="plain")
            else:
                renderable = {
                    'environment': credentials.get('environment'),
                    'api_key': credentials.get('api_key')
                }
                self.app.render(renderable, format=OutputFormat.JSON.value)
