from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.core.version import get_version
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class Enterprise(Controller):
    class Meta:
        label = 'enterprise'

        # text displayed at the top of --help output
        description = 'enterprise controller is used for enterprise related commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli enterprise'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='show command used to showing enterprise details',
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
        enterprise_client = APIClient(db.get_configure()).get_enterprise_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            response = enterprise_client.get_enterprise(enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an enterprise: {e}")
            self.app.log.error(f"Failed to show details of an enterprise, reason: {e.reason}")
            return

        address = None
        location = None
        zipcode = None
        email = None
        if response.details:
            address = response.details.registered_address
            location = response.details.location
            zipcode = response.details.zipcode
            email = response.details.contact_email

        if not self.app.pargs.json:
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [
                {title: 'id', details: response.id},
                {title: 'name', details: response.name},
                {title: 'short_code', details: response.short_code},
                {title: 'address', details: address},
                {title: 'location', details: location},
                {title: 'zipcode', details: zipcode},
                {title: 'email', details: email}
            ]
            print(white(f"\tENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = {
                'id': response.id,
                'name': response.name,
                'short_code': response.short_code,
                'address': address,
                'location': location,
                'zipcode': zipcode,
                'email': email
            }
            print(white(f"ENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)
