from http import HTTPStatus

from cement import Controller, ex
from esperclient.rest import ApiException

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class ApplicationVersion(Controller):
    class Meta:
        label = 'version'

        # text displayed at the top of --help output
        description = 'Version commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli version'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='List application versions',
        arguments=[
            (['-a', '--app'],
             {'help': 'Application id',
              'action': 'store',
              'dest': 'application'}),
            (['-c', '--code'],
             {'help': 'Version code',
              'action': 'store',
              'dest': 'version_code'}),
            (['-n', '--number'],
             {'help': 'Build number',
              'action': 'store',
              'dest': 'build_number'}),
            (['-lf', '--legacy-format'],
             {'help': 'Use legacy format, choices are [true, false]',
              'action': 'store',
              'choices': ['true', 'false'],
              'default': 'true',
              'dest': 'legacy_format'}),
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
        """Command to list application versions"""
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.application:
            application_id = self.app.pargs.application
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.debug('[version-list] There is no active application.')
                self.app.render('There is no active application.\n')
                return

            application_id = application.get('id')

        version_code = self.app.pargs.version_code
        build_number = self.app.pargs.build_number
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset
        
        # Get legacy_format flag and convert to boolean
        legacy_format_str = getattr(self.app.pargs, 'legacy_format', 'true')
        legacy_format = (legacy_format_str == 'true')

        kwargs = {}
        if version_code:
            kwargs['version_code'] = version_code

        if build_number:
            kwargs['build_number'] = build_number

        try:
            # Find application versions in an enterprise
            response = application_client.get_app_versions(application_id, enterprise_id, limit=limit, offset=offset,
                                                           legacy_format=legacy_format, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[version-list] Failed to list applications: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        self.app.render(f"Total Number of Versions: {response.count}")
        if not self.app.pargs.json:
            versions = []

            # Determine which columns to show based on legacy_format
            if legacy_format:
                # Legacy format: show BUILD NUMBER
                label = {
                    'id': "ID",
                    'version_code': "VERSION CODE",
                    'build_number': "BUILD NUMBER",
                    'size_in_mb': "SIZE IN MB",
                    'release_track': "RELEASE TRACK",
                    'installed_count': "INSTALLED COUNT",
                }
            else:
                # New format: show VERSION NAME
                label = {
                    'id': "ID",
                    'version_name': "VERSION NAME",
                    'version_code': "VERSION CODE",
                    'size_in_mb': "SIZE IN MB",
                    'release_track': "RELEASE TRACK",
                    'installed_count': "INSTALLED COUNT",
                }

            for version in response.results:
                if legacy_format:
                    # Legacy format: use build_number
                    version_dict = {
                        label['id']: version.id,
                        label['version_code']: version.version_code,
                        label['build_number']: version.build_number if hasattr(version, 'build_number') and version.build_number else '',
                        label['size_in_mb']: version.size_in_mb,
                        label['release_track']: version.release_track,
                        label['installed_count']: version.installed_count if version.installed_count else 0
                    }
                else:
                    # New format: use version_name
                    version_dict = {
                        label['id']: version.id,
                        label['version_name']: version.version_name if hasattr(version, 'version_name') and version.version_name else '',
                        label['version_code']: version.version_code,
                        label['size_in_mb']: version.size_in_mb,
                        label['release_track']: version.release_track,
                        label['installed_count']: version.installed_count if version.installed_count else 0
                    }
                versions.append(version_dict)
            self.app.render(versions, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            versions = []
            for version in response.results:
                if legacy_format:
                    # Legacy format: include build_number
                    version_dict = {
                        'id': version.id,
                        'version_code': version.version_code,
                        'build_number': version.build_number if hasattr(version, 'build_number') and version.build_number else None,
                        'size_in_mb': version.size_in_mb,
                        'release_track': version.release_track,
                        'installed_count': version.installed_count if version.installed_count else 0
                    }
                else:
                    # New format: include version_name
                    version_dict = {
                        'id': version.id,
                        'version_name': version.version_name if hasattr(version, 'version_name') and version.version_name else None,
                        'version_code': version.version_code,
                        'size_in_mb': version.size_in_mb,
                        'release_track': version.release_track,
                        'installed_count': version.installed_count if version.installed_count else 0
                    }
                versions.append(version_dict)
            self.app.render(versions, format=OutputFormat.JSON.value)

    def _version_basic_response(self, version, format=OutputFormat.TABULATED, legacy_format=True):
        # Include both version_name and build_number in valid_keys for flexibility
        valid_keys = ['id', 'version_code', 'version_name', 'build_number', 'size_in_mb', 'release_track']

        if format == OutputFormat.JSON:
            renderable = {k: v for k, v in version.to_dict().items() if k in valid_keys}
            if version:
                renderable['installed_count'] = version.installed_count if version.installed_count else 0
        else:
            title = "TITLE"
            details = "DETAILS"
            # Filter based on legacy_format
            if legacy_format:
                # Legacy format: prioritize build_number
                filtered_dict = {k: v for k, v in version.to_dict().items() if k in valid_keys and k != 'version_name'}
            else:
                # New format: prioritize version_name
                filtered_dict = {k: v for k, v in version.to_dict().items() if k in valid_keys and k != 'build_number'}
            
            renderable = [{title: k, details: v} for k, v in filtered_dict.items()]

            if version:
                renderable.append(
                    {
                        title: 'installed_count',
                        details: version.installed_count if version.installed_count else 0
                    }
                )
        return renderable

    @ex(
        help='Show application version details',
        arguments=[
            (['version_id'],
             {'help': 'Version id',
              'action': 'store'}),
            (['-a', '--app'],
             {'help': 'Application id',
              'action': 'store',
              'dest': 'application'}),
            (['-lf', '--legacy-format'],
             {'help': 'Use legacy format, choices are [true, false]',
              'action': 'store',
              'choices': ['true', 'false'],
              'default': 'true',
              'dest': 'legacy_format'}),
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

        if self.app.pargs.application:
            application_id = self.app.pargs.application
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.debug('[version-show] There is no active application.')
                self.app.render('There is no active application.\n')
                return

            application_id = application.get('id')

        # Get legacy_format flag and convert to boolean
        legacy_format_str = getattr(self.app.pargs, 'legacy_format', 'true')
        legacy_format = (legacy_format_str == 'true')

        try:
            response = application_client.get_app_version(version_id, application_id, enterprise_id, legacy_format=legacy_format)
        except ApiException as e:
            self.app.log.error(f"[version-show] Failed to show details of an version: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        if not self.app.pargs.json:
            renderable = self._version_basic_response(response, format=OutputFormat.TABULATED, legacy_format=legacy_format)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._version_basic_response(response, format=OutputFormat.JSON, legacy_format=legacy_format)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Delete application version',
        arguments=[
            (['version_id'],
             {'help': 'Version id',
              'action': 'store'}),
            (['-a', '--app'],
             {'help': 'Application id',
              'action': 'store',
              'dest': 'application'}),
        ]
    )
    def delete(self):
        version_id = self.app.pargs.version_id

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        if self.app.pargs.application:
            application_id = self.app.pargs.application
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.debug('[version-delete] There is no active application.')
                self.app.render('There is no active application.\n')
                return

            application_id = application.get('id')

        try:
            application_client.delete_app_version(version_id, application_id, enterprise_id)
            self.app.log.debug(f"[version-delete] Version with id : {version_id} deleted successfully")
            self.app.render(f"Version with id {version_id} deleted successfully\n")
        except ApiException as e:
            self.app.log.error(f"[version-delete] Failed to delete a version: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return

        # Unset current application if matching
        try:
            application_client.get_application(application_id, enterprise_id)
        except ApiException as e:
            if e.status == HTTPStatus.NOT_FOUND:
                application = db.get_application()
                if application and application.get('id') and application_id == application.get('id'):
                    db.unset_application()
                    self.app.log.debug(f'[version-delete] Unset the active application {application_id}')
            else:
                self.app.log.debug(f"[version-delete] Failed to get an application when deleting a version: {e}")
                self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")


    @ex(
        help='List devices with the app version installed',
        arguments=[
            (['version_id'],
             {'help': 'Version id',
              'action': 'store'}),
            (['-a', '--app'],
             {'help': 'Application id',
              'action': 'store',
              'dest': 'application'}),
            (['-s', '--search'],
             {'help': 'A search term',
              'action': 'store',
              'dest': 'search'}),
            (['-l', '--limit'],
             {'help': 'Number of results to return per page',
              'action': 'store',
              'default': 20,
              'dest': 'limit'}),
            (['-o', '--offset'],
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
    def devices(self):

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        application_client = APIClient(db.get_configure()).get_application_api_client()
        enterprise_id = db.get_enterprise_id()

        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if self.app.pargs.search:
            kwargs['search'] = self.app.pargs.search

        if self.app.pargs.application:
            application_id = self.app.pargs.application
        else:
            application = db.get_application()
            if not application or not application.get('id'):
                self.app.log.debug('[version-devices] There is no active application.')
                self.app.render('There is no active application.\n')
                return

            application_id = application.get('id')

        if self.app.pargs.version_id:
            version_id = self.app.pargs.version_id
        else:
            self.app.log.debug('version id is not given')
            self.app.render('version id is not given\n')
            return 

        try:
            response = application_client.get_install_devices(version_id, application_id, enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[version-devices] Failed to list devices: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return
        
        self.app.render(f"Total Number of Devices: {response.count}\n")
        if not self.app.pargs.json:
            devices = []

            label = {
                'id': "ID",
                'device_name': "DEVICE NAME",
                'alias_name': "ALIAS NAME",
                'group_name': "GROUP NAME",
            }

            for device in response.results:
                devices.append(
                    {
                        label['id']: device.id,
                        label['device_name']: device.device_name,
                        label['alias_name']: device.alias_name,
                        label['group_name']: device.group_name,
                    }
                )
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            devices = []
            for device in response.results:
                devices.append(
                    {
                        'id': device.id,
                        'device_name': device.device_name,
                        'alias_name': device.alias_name,
                        'group_name': device.group_name,
                    }
                )
            self.app.render(devices, format=OutputFormat.JSON.value)