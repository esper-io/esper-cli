from cement import Controller, ex
from cement.utils.version import get_version_banner
from crayons import white
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.core.version import get_version
from esper.ext.utils import validate_creds_exists, get_client_config, get_application_api_instance

VERSION_BANNER = """
Command Line Tool for Esper SDK %s
%s
""" % (get_version(), get_version_banner())


class ApplicationVersion(Controller):
    class Meta:
        label = 'ApplicationVersion'

        # text displayed at the top of --help output
        description = 'ApplicationVersion command to list, show and delete version'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli application version'

        stacked_type = 'nested'
        stacked_on = 'application'

    @ex(
        help='version-list command to list application versions',
        arguments=[
            (['-a', '--application'],
             {'help': 'Filter versions by application id',
              'action': 'store',
              'dest': 'application_id'}),
            (['-vc', '--versioncode'],
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
    def version_list(self):
        """Command to list application versions"""
        try:

            validate_creds_exists(self.app)
            config = get_client_config(self.app)
            api_instance = get_application_api_instance(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            application_id = self.app.pargs.application_id
            version_code = self.app.pargs.version_code
            build_number = self.app.pargs.build_number
            limit = self.app.pargs.limit
            offset = self.app.pargs.offset

            kwargs = {}
            if version_code:
                kwargs['version_code'] = version_code

            if build_number:
                kwargs['build_number'] = build_number

            # Find applications in an enterprise
            response = api_instance.get_app_versions(application_id, enterprise_id, limit=limit, offset=offset,
                                                     **kwargs)

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

        except ApiException as e:
            self.app.log.debug(f"Failed to list applications: {e}")
            self.app.log.error(f"Failed to list applications, reason: {e.reason}")

    @ex(
        help='version-show command to showing application version specific details',
        arguments=[
            (['version_id'],
             {'help': 'Show details about the version by id',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def version_show(self):
        try:
            version_id = self.app.pargs.version_id

            validate_creds_exists(self.app)
            config = get_client_config(self.app)
            api_instance = get_application_api_instance(config)
            enterprise_id = self.app.creds.all()[0].get("enterprise")

            response = api_instance.get_app_version(version_id, enterprise_id)

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