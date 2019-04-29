from http import HTTPStatus

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


class ApplicationVersion(Controller):
    class Meta:
        label = 'version'

        # text displayed at the top of --help output
        description = 'version controller is used for version related commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli version'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='list command used to list application versions',
        arguments=[
            (['-a', '--app'],
             {'help': 'Filter versions by application id',
              'action': 'store',
              'dest': 'application_id'}),
            (['-vc', '--version_code'],
             {'help': 'Filter versions by version code',
              'action': 'store',
              'dest': 'version_code'}),
            (['-bn', '--build_number'],
             {'help': 'Filter versions by build number',
              'action': 'store',
              'dest': 'build_number'}),
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
    def list(self):
        """Command to list application versions"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.application_id:
            application_id = self.app.pargs.application_id
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.info('Not set the current application.')
                return

            application_id = application.get('id')

        version_code = self.app.pargs.version_code
        build_number = self.app.pargs.build_number
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if version_code:
            kwargs['version_code'] = version_code

        if build_number:
            kwargs['build_number'] = build_number

        try:
            # Find application versions in an enterprise
            response = application_client.get_app_versions(application_id, enterprise_id, limit=limit, offset=offset,
                                                           **kwargs)
        except ApiException as e:
            self.app.log.debug(f"Failed to list applications: {e}")
            self.app.log.error(f"Failed to list applications, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            versions = []

            label = {
                'id': white("ID", bold=True),
                'version_code': white("VERSION CODE", bold=True),
                'build_number': white("BUILD NUMBER", bold=True),
                'size_in_mb': white("SIZE IN Mb", bold=True),
                'release_track': white("RELEASE TRACK", bold=True),
                'installed_count': white("INSTALLED COUNT", bold=True),
            }

            for version in response.results:
                versions.append(
                    {
                        label['id']: version.id,
                        label['version_code']: version.version_code,
                        label['build_number']: version.build_number,
                        label['size_in_mb']: version.size_in_mb,
                        label['release_track']: version.release_track,
                        label['installed_count']: version.installed_count if version.installed_count else 0
                    }
                )
            print(white(f"\tTotal Number of Versions: {response.count}", bold=True))
            self.app.render(versions, format=OutputFormat.TABULATED.value, headers="keys",
                            tablefmt="fancy_grid")
        else:
            versions = []
            for version in response.results:
                versions.append(
                    {
                        'id': version.id,
                        'version_code': version.version_code,
                        'build_number': version.build_number,
                        'size_in_mb': version.size_in_mb,
                        'release_track': version.release_track,
                        'installed_count': version.installed_count if version.installed_count else 0
                    }
                )
            print(white(f"Total Number of Versions: {response.count}", bold=True))
            self.app.render(versions, format=OutputFormat.JSON.value)

    def _version_basic_response(self, version, format=OutputFormat.TABULATED):
        valid_keys = ['id', 'version_code', 'build_number', 'size_in_mb', 'release_track']

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in version.to_dict().items() if k in valid_keys}
            if version:
                renderable['installed_count'] = version.installed_count if version.installed_count else 0
        else:
            title = white("TITLE", bold=True)
            details = white("DETAILS", bold=True)
            renderable = [{title: k, details: v} for k, v in version.to_dict().items() if k in valid_keys]

            if version:
                renderable.append(
                    {
                        title: 'installed_count',
                        details: version.installed_count if version.installed_count else 0
                    }
                )
        return renderable

    @ex(
        help='show command used to showing application version specific details',
        arguments=[
            (['version_id'],
             {'help': 'Show details about the version by id',
              'action': 'store'}),
            (['-a', '--application'],
             {'help': 'Application id of version',
              'action': 'store',
              'dest': 'application_id'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def show(self):
        version_id = self.app.pargs.version_id

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.application_id:
            application_id = self.app.pargs.application_id
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.info('Not set the current application.')
                return

            application_id = application.get('id')

        try:
            response = application_client.get_app_version(version_id, application_id, enterprise_id)
        except ApiException as e:
            self.app.log.debug(f"Failed to show details of an version: {e}")
            self.app.log.error(f"Failed to show details of an version, reason: {e.reason}")
            return

        if not self.app.pargs.json:
            print(white(f"\tVERSION DETAILS of {version_id}", bold=True))
            renderable = self._version_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="fancy_grid")
        else:
            print(white(f"VERSION DETAILS of {version_id}", bold=True))
            renderable = self._version_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='delete command used to delete particular application version',
        arguments=[
            (['version_id'],
             {'help': 'Delete a version by id',
              'action': 'store'}),
            (['-a', '--app'],
             {'help': 'Application id of version',
              'action': 'store',
              'dest': 'application_id'}),
        ]
    )
    def delete(self):
        version_id = self.app.pargs.version_id

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.application_id:
            application_id = self.app.pargs.application_id
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.info('Not set the current application.')
                return

            application_id = application.get('id')

        try:
            application_client.delete_app_version(version_id, application_id, enterprise_id)
            self.app.log.info(f"Version with id : {version_id} deleted successfully")
        except ApiException as e:
            self.app.log.debug(f"Failed to delete a version: {e}")
            self.app.log.error(f"Failed to delete a version, reason: {e.reason}")
            return

        # Unset current application if matching
        try:
            application_client.get(application_id, enterprise_id)
        except ApiException as e:
            if e.status == HTTPStatus.NOT_FOUND:
                application = db.get_application()
                if application and application.get('id') and application_id == application.get('id'):
                    db.unset_application()
                    self.app.log.debug(f'Unset the current application {application_id}')
            else:
                self.app.log.debug(f"Failed to get an application when deleting a version: {e}")
                self.app.log.warning(f"Failed to get an application when deleting a version, reason: {e.reason}")
