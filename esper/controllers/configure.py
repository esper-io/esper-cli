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

            (['--json'],
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
            api_key = prompt.query("Esper API Key: ")
            tenant = input("Tenant name: ")

            enterprise_client = APIClient({'api_key': api_key, 'tenant': tenant}).get_enterprise_api_client()
            try:
                response = enterprise_client.get_all_enterprises()
            except ApiException as e:
                self.app.log.error(f"[configure] Failed to list enterprises: {e}")
                if e.status == HTTPStatus.UNAUTHORIZED:
                    self.app.render("You are not authorized, invalid API Key.")
                else:
                    error_message = json.loads(e.body).get('message') if e.body and json.loads(e.body).get(
                        'message') else e.reason
                    self.app.render(f"ERROR: {error_message}")
                return

            if response.results and len(response.results) > 0:
                enterprise_id = response.results[0].id
            else:
                self.app.log.info(f"[configure] API key is not associated with any enterprise.")
                self.app.render("API key is not associated with any enterprise.")
                return

            credentials = {
                "api_key": api_key,
                "enterprise_id": enterprise_id,
                "tenant": tenant
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
                    {title: 'api_key', details: credentials.get('api_key')},
                    {title: 'tenant', details: credentials.get('tenant')}
                ]
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys",
                                tablefmt="plain")
            else:
                renderable = {
                    'api_key': credentials.get('api_key'),
                    'tenant': credentials.get('tenant')
                }
                self.app.render(renderable, format=OutputFormat.JSON.value)
