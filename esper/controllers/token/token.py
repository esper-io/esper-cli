from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class Token(Controller):
    class Meta:
        label = 'token'

        # text displayed at the top of --help output
        description = 'token command for displaying all the information associated with the token'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli token'

        stacked_type = 'nested'
        stacked_on = 'base'


    def _token_basic_response(self, token, format=OutputFormat.TABULATED):
        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'Enterprise Id', details: token.enterprise},
                {title: 'Token', details: token.token},
                {title: 'Expires On', details: token.expires_on},
                {title: 'Scope', details: token.scope},
                {title: 'Created On', details: token.created_on},
                {title: 'Updated On', details: token.updated_on}
            ]
        else:
            renderable = {
                'Enterprise': token.enterprise,
                'Developer App': token.developer_app,
                'Token': token.token,
                'Expires On': token.scope,
                'Created On': str(token.created_on),
                'Updated On': str(token.updated_on)
            }

        return renderable



    @ex(
        help='Show token details',
        arguments=[
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        token_client = APIClient(db.get_configure()).get_token_api_client()
        response = None
        try:
            response = token_client.get_token_info()
        except ApiException as e:
            self.app.log.error(f"[token-show] Failed to show details of an token: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._token_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._token_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
