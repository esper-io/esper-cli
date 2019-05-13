from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists


class Application(Controller):
    class Meta:
        label = 'app'

        # text displayed at the top of --help output
        description = 'Application commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli app'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List applications',
        arguments=[
            (['-n', '--name'],
             {'help': 'Application name',
              'action': 'store',
              'dest': 'name'}),
            (['-p', '--package'],
             {'help': 'Package name',
              'action': 'store',
              'dest': 'package'}),
            (['-l', '--limit'],
             {'help': 'Number of results to return per page',
              'action': 'store',
              'default': 20,
              'dest': 'limit'}),
            (['-i', '--offset'],
             {'help': 'The initial index from which to return the results',
              'action': 'store',
              'default': 0,
              'dest': 'offset'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def list(self):
        """Command to list applications"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        name = self.app.pargs.name
        package = self.app.pargs.package
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if name:
            kwargs['application_name'] = name

        if package:
            kwargs['package_name'] = package

        kwargs['is_hidden'] = False

        try:
            # Find applications in an enterprise
            response = application_client.get_all_applications(enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.debug(f"Failed to list applications: {e}")
            self.app.log.error(f"Failed to list applications, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            applications = []

            label = {
                'id': "ID",
                'name': "NAME",
                'package': "PACKAGE NAME",
                'version_count': "NO. OF VERSIONS"
            }

            for application in response.results:
                applications.append(
                    {
                        label['id']: application.id,
                        label['name']: application.application_name,
                        label['package']: application.package_name,
                        label['version_count']: len(application.versions) if application.versions else 0
                    }
                )
            print(f"Total Number of Applications: {response.count}")
            self.app.render(applications, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            applications = []
            for application in response.results:
                applications.append(
                    {
                        'id': application.id,
                        'name': application.application_name,
                        'package': application.package_name,
                        'version_count': len(application.versions) if application.versions else 0
                    }
                )
            print(f"Total Number of Applications: {response.count}")
            self.app.render(applications, format=OutputFormat.JSON.value)

    def _application_basic_response(self, application, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'application_name', 'package_name', 'developer', 'category', 'content_rating',
                      'compatibility']

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in application.to_dict().items() if k in valid_keys}
            if application:
                renderable['version_count'] = len(application.versions)
        else:
            title = "TITLE"
            details = "DETAILS"
            renderable = [{title: k, details: v} for k, v in application.to_dict().items() if k in valid_keys]

            if application:
                renderable.append({title: 'version_count', details: len(application.versions)})
        return renderable

    @ex(
        help='Show application details and set active application',
        arguments=[
            (['application_id'],
             {'help': 'Application id',
              'action': 'store'}),
            (['-a', '--active'],
             {'help': 'Set application as active application for further application specific commands',
              'action': 'store_true',
              'dest': 'active'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        application_id = self.app.pargs.application_id
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            response = application_client.get_application(application_id, enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an application: {e}")
            self.app.log.error(f"Failed to show details of an application, reason: {e.reason}")
            return

        if self.app.pargs.active:
            db.set_application({'id': application_id})

        if not self.app.pargs.json:
            renderable = self._application_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._application_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Upload application',
        arguments=[
            (['application_file'],
             {'help': 'Application file',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def upload(self):
        application_file = self.app.pargs.application_file

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            response = application_client.upload(enterprise_id, enterprise_id, application_file)
        except ApiException as e:
            self.app.log.debug(f"Failed to upload an application: {e}")
            self.app.log.error(f"Failed to upload an application, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            renderable = self._application_basic_response(response.application)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._application_basic_response(response.application, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Delete application',
        arguments=[
            (['application_id'],
             {'help': 'Delete an application by id',
              'action': 'store'}),
        ]
    )
    def delete(self):
        application_id = self.app.pargs.application_id

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        try:
            application_client.delete_application(application_id, enterprise_id)
            self.app.log.debug(f"Application with id : {application_id} deleted successfully")
            print(f"Application with id {application_id} deleted successfully")

            # Unset current application if matching
            application = db.get_application()
            if application and application.get('id') and application_id == application.get('id'):
                db.unset_application()
                self.app.log.debug(f'Unset the active application {application_id}')
        except ApiException as e:
            self.app.log.debug(f"Failed to delete an application: {e}")
            self.app.log.error(f"Failed to delete an application, reason: {e.reason}")
            return

    @ex(
        help='Set, show or unset the active application',
        arguments=[
            (['-i', '--id'],
             {'help': 'Application id.',
              'action': 'store',
              'dest': 'id'}),
            (['-u', '--unset'],
             {'help': 'Unset the active application',
              'action': 'store_true',
              'dest': 'unset'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.id:
            application_id = self.app.pargs.id
            try:
                response = application_client.get_application(application_id, enterprise_id)
                db.set_application({'id': application_id})
            except ApiException as e:
                self.app.log.debug(f"Failed to show active application: {e}")
                self.app.log.error(f"Failed to set active application, reason: {e.reason}")
                return
        elif self.app.pargs.unset:
            application = db.get_application()
            if application is None or application.get('id') is None:
                print('There is no active application.')
                return

            db.unset_application()
            self.app.log.debug(f"Unset the active application {application.get('id')}")
            print(f"Unset the active application {application.get('id')}")
            return
        else:
            application = db.get_application()
            if application is None or application.get('id') is None:
                print('There is no active application.')
                return

            application_id = application.get('id')
            try:
                response = application_client.get_application(application_id, enterprise_id)
                db.set_application({'id': application_id})
            except ApiException as e:
                self.app.log.debug(f"Failed to show active application: {e}")
                self.app.log.error(f"Failed to show active application, reason: {e.reason}")
                return

        if not self.app.pargs.json:
            renderable = self._application_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._application_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)
