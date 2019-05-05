from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient import Enterprise as EnterpriseRequest
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

    def _enterprise_basic_response(self, enterprise, format=OutputFormat.TABULATED):
        registered_name = None
        address = None
        location = None
        zipcode = None
        email = None
        contact_person = None
        contact_number = None
        if enterprise.details:
            registered_name = enterprise.details.registered_name
            address = enterprise.details.registered_address
            location = enterprise.details.location
            zipcode = enterprise.details.zipcode
            email = enterprise.details.contact_email
            contact_person = enterprise.details.contact_person
            contact_number = enterprise.details.contact_number

        if format == OutputFormat.TABULATED:
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [
                {title: 'id', details: enterprise.id},
                {title: 'name', details: enterprise.name},
                {title: 'display_name', details: enterprise.display_name},
                {title: 'registered_name', details: registered_name},
                {title: 'address', details: address},
                {title: 'location', details: location},
                {title: 'zipcode', details: zipcode},
                {title: 'email', details: email},
                {title: 'contact_person', details: contact_person},
                {title: 'contact_number', details: contact_number}
            ]
        else:
            renderable = {
                'id': enterprise.id,
                'name': enterprise.name,
                'display_name': enterprise.display_name,
                'registered_name': registered_name,
                'address': address,
                'location': location,
                'zipcode': zipcode,
                'email': email,
                'contact_person': contact_person,
                'contact_number': contact_number
            }

        return renderable

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

        if not self.app.pargs.json:
            renderable = self._enterprise_basic_response(response)
            print(white(f"\tENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._enterprise_basic_response(response, OutputFormat.JSON)
            print(white(f"ENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='update command used to update enterprise details',
        arguments=[
            (['-n', '--name'],
             {'help': 'Enterprise name',
              'action': 'store',
              'dest': 'name'}),
            (['-dn', '--dispname'],
             {'help': 'Display name',
              'action': 'store',
              'dest': 'display_name'}),
            (['-rn', '--regname'],
             {'help': 'Registered name',
              'action': 'store',
              'dest': 'registered_name'}),
            (['-a', '--address'],
             {'help': 'Enterprise address',
              'action': 'store',
              'dest': 'address'}),
            (['-l', '--location'],
             {'help': 'Enterprise location',
              'action': 'store',
              'dest': 'location'}),
            (['-z', '--zipcode'],
             {'help': 'Zip code',
              'action': 'store',
              'dest': 'zipcode'}),
            (['-e', '--email'],
             {'help': 'Contact email',
              'action': 'store',
              'dest': 'email'}),
            (['-p', '--person'],
             {'help': 'Contact person',
              'action': 'store',
              'dest': 'contact_person'}),
            (['-cn', '--number'],
             {'help': 'Contact number',
              'action': 'store',
              'dest': 'contact_number'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def update(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        enterprise_client = APIClient(db.get_configure()).get_enterprise_api_client()
        enterprise_id = db.get_enterprise_id()
        data = EnterpriseRequest()

        if self.app.pargs.name:
            data.name = self.app.pargs.name

        if self.app.pargs.display_name:
            data.display_name = self.app.pargs.display_name

        details = {}
        if self.app.pargs.registered_name:
            details['registered_name'] = self.app.pargs.registered_name

        if self.app.pargs.address:
            details['registered_address'] = self.app.pargs.address

        if self.app.pargs.location:
            details['location'] = self.app.pargs.location

        if self.app.pargs.email:
            details['contact_email'] = self.app.pargs.email

        if self.app.pargs.contact_person:
            details['contact_person'] = self.app.pargs.contact_person

        if self.app.pargs.contact_number:
            details['contact_number'] = self.app.pargs.contact_number

        if bool(details):
            data.details = details

        try:
            response = enterprise_client.partial_update_enterprise(enterprise_id, data)
        except ApiException as e:
            self.app.log.debug(f"Failed to update details of an enterprise: {e}")
            self.app.log.error(f"Failed to update details of an enterprise, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._enterprise_basic_response(response)
            print(white(f"\tENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            renderable = self._enterprise_basic_response(response, OutputFormat.JSON)
            print(white(f"ENTERPRISE DETAILS", bold=True))
            self.app.render(renderable, format=OutputFormat.JSON.value)
