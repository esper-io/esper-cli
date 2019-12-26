from cement import Controller, ex
from esperclient import EnterpriseUpdateV1
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class Enterprise(Controller):
    class Meta:
        label = 'enterprise'

        # text displayed at the top of --help output
        description = 'Enterprise commands'

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
        if enterprise:
            registered_name = enterprise.registered_name
            address = enterprise.registered_address
            location = enterprise.location
            zipcode = enterprise.zipcode
            email = enterprise.contact_email
            contact_person = enterprise.contact_person
            contact_number = enterprise.contact_number

        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'Enterprise Id', details: enterprise.id},
                {title: 'Name', details: enterprise.name},
                {title: 'Registered Name', details: registered_name},
                {title: 'Address', details: address},
                {title: 'Location', details: location},
                {title: 'Zip Code', details: zipcode},
                {title: 'Email', details: email},
                {title: 'Contact Person', details: contact_person},
                {title: 'Contact Number', details: contact_number}
            ]
        else:
            renderable = {
                'Enterprise Id': enterprise.id,
                'Name': enterprise.name,
                'Registered Name': registered_name,
                'Address': address,
                'Location': location,
                'Zip Code': zipcode,
                'Email': email,
                'Contact Person': contact_person,
                'Contact Number': contact_number
            }

        return renderable

    @ex(
        help='Show enterprise details',
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
            self.app.log.error(f"[enterprise-show] Failed to show details of an enterprise: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._enterprise_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._enterprise_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Update enterprise details',
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
        data = EnterpriseUpdateV1()
        update_dict = {}

        if self.app.pargs.name:
            update_dict['name'] = self.app.pargs.name

        if self.app.pargs.registered_name:
            update_dict['registered_name'] = self.app.pargs.registered_name

        if self.app.pargs.address:
            update_dict['registered_address'] = self.app.pargs.address

        if self.app.pargs.location:
            update_dict['location'] = self.app.pargs.location

        if self.app.pargs.zipcode:
            update_dict['zipcode'] = self.app.pargs.zipcode

        if self.app.pargs.email:
            update_dict['contact_email'] = self.app.pargs.email

        if self.app.pargs.contact_person:
            update_dict['contact_person'] = self.app.pargs.contact_person

        if self.app.pargs.contact_number:
            update_dict['contact_number'] = self.app.pargs.contact_number

        try:
            response = enterprise_client.partial_update_enterprise(enterprise_id, update_dict)
        except ApiException as e:
            self.app.log.error(f"[enterprise-update] Failed to update details of an enterprise: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._enterprise_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._enterprise_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
