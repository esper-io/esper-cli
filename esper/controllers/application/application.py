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


class Application(Controller):
    class Meta:
        label = 'application'

        # text displayed at the top of --help output
        # description = 'Applications command to list and show application details'

        # text displayed at the bottom of --help output
        # epilog = 'Usage: espercli applications'

        stacked_type = 'embedded'
        stacked_on = 'base'

    @ex(
        help='application-list command to list application details',
        arguments=[
            (['-n', '--name'],
             {'help': 'Filter devices by application name',
              'action': 'store',
              'dest': 'name'}),
            (['-p', '--package'],
             {'help': 'Filter devices by package name',
              'action': 'store',
              'dest': 'package'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
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
        ]
    )
    def application_list(self):
        """Command to list applications"""
        try:

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

            # Find applications in an enterprise
            response = application_client.list(enterprise_id, limit=limit, offset=offset, **kwargs)

            if not self.app.pargs.json:
                applications = []

                label = {
                    'id': white("ID", bold=True),
                    'name': white("NAME", bold=True),
                    'package': white("PACKAGE NAME", bold=True),
                    'version_count': white("NO. OF VERSIONS", bold=True)
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
                print(white(f"\tTotal Number of Applications: {response.count}", bold=True))
                self.app.render(applications, format=OutputFormat.TABULATED.value, headers="keys",
                                tablefmt="fancy_grid")
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
                print(white(f"Total Number of Applications: {response.count}", bold=True))
                self.app.render(applications, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.debug(f"Failed to list applications: {e}")
            self.app.log.error(f"Failed to list applications, reason: {e.reason}")

    def _application_basic_response(self, application, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'application_name', 'package_name', 'developer', 'category', 'content_rating',
                      'compatibility']

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in application.to_dict().items() if k in valid_keys}
            if application:
                renderable['version_count'] = len(application.versions)
        else:
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [{title: k, details: v} for k, v in application.to_dict().items() if k in valid_keys]

            if application:
                renderable.append({title: 'version_count', details: len(application.versions)})
        return renderable

    @ex(
        help='application-show command to showing application specific details',
        arguments=[
            (['application_id'],
             {'help': 'Show details about the application by id',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
            (['-s', '--set'],
             {'help': 'Set application as current application for further application related commands',
              'action': 'store_true',
              'dest': 'set'})
        ]
    )
    def application_show(self):
        try:
            validate_creds_exists(self.app)
            db = DBWrapper(self.app.creds)

            application_id = self.app.pargs.application_id

            if self.app.pargs.set:
                db.set_application({'id': application_id})

            application_client = APIClient(db.get_configure()).get_application_api_client()
            enterprise_id = db.get_enterprise_id()

            response = application_client.get(application_id, enterprise_id)

            if not self.app.pargs.json:
                print(white(f"\tAPPLICATION DETAILS of {response.application_name}", bold=True))
                renderable = self._application_basic_response(response)
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                renderable = self._application_basic_response(response, OutputFormat.JSON)
                self.app.render(renderable, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an application: {e}")
            self.app.log.error(f"Failed to show details of an application, reason: {e.reason}")

    @ex(
        help='Upload application file',
        arguments=[
            (['application_file'],
             {'help': 'Application file path to upload',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def application_upload(self):
        try:
            application_file = self.app.pargs.application_file

            validate_creds_exists(self.app)
            db = DBWrapper(self.app.creds)
            application_client = APIClient(db.get_configure()).get_application_upload_api_client()
            enterprise_id = db.get_enterprise_id()

            response = application_client.upload(enterprise_id, enterprise_id, application_file)

            if not self.app.pargs.json:
                print(white(f"\tAPPLICATION DETAILS of {response.application.application_name}", bold=True))
                renderable = self._application_basic_response(response.application)
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                renderable = self._application_basic_response(response.application, OutputFormat.JSON)
                self.app.render(renderable, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.debug(f"Failed to upload an application: {e}")
            self.app.log.error(f"Failed to upload an application, reason: {e.reason}")

    @ex(
        help='application-delete command to delete particular an application',
        arguments=[
            (['application_id'],
             {'help': 'Show details about the application by id',
              'action': 'store'}),
        ]
    )
    def application_delete(self):
        try:
            application_id = self.app.pargs.application_id

            validate_creds_exists(self.app)
            db = DBWrapper(self.app.creds)
            application_client = APIClient(db.get_configure()).get_application_api_client()
            enterprise_id = db.get_enterprise_id()

            application_client.delete(application_id, enterprise_id)
            self.app.log.info(f"Application with id : {application_id} deleted successfully")
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an application: {e}")
            self.app.log.error(f"Failed to show details of an application, reason: {e.reason}")

    @ex(
        help='application-current command to show or unset the current application',
        arguments=[
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
            (['-u', '--unset'],
             {'help': 'Unset the current application',
              'action': 'store_true',
              'dest': 'unset'})
        ]
    )
    def application_current(self):
        try:
            validate_creds_exists(self.app)
            db = DBWrapper(self.app.creds)

            application_id = None
            application = db.get_application()
            if application:
                application_id = application.get('id', None)

            if self.app.pargs.unset:
                if not application_id:
                    self.app.log.info('Not set the current application.')
                    return

                db.unset_application()
                self.app.log.info(f'Unset the current application {application_id}')
                return

            if not application_id:
                self.app.log.error("Application id is empty!")
                return

            application_client = APIClient(db.get_configure()).get_application_api_client()
            enterprise_id = db.get_enterprise_id()

            response = application_client.get(application_id, enterprise_id)

            if not self.app.pargs.json:
                print(white(f"\tAPPLICATION DETAILS of {response.application_name}", bold=True))
                renderable = self._application_basic_response(response)
                self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
            else:
                renderable = self._application_basic_response(response, OutputFormat.JSON)
                self.app.render(renderable, format=OutputFormat.JSON.value)

        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an application: {e}")
            self.app.log.error(f"Failed to show details of an application, reason: {e.reason}")
