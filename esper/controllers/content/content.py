import os
import random
import time
from tqdm import tqdm
from pathlib import Path

from cement import ex, Controller
from esperclient.rest import ApiException
from esperclient import Content

from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message


class Content(Controller):
    class Meta:
        label = 'content'

        # text displayed at the top of --help output
        description = 'Content management'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli content'

        stacked_type = 'nested'
        stacked_on = 'base'


    @ex(
        help='List all contents',
        arguments=[
            (['-s', '--search'],
             {'help': 'Searches by name/tags/description',
              'action': 'store',
              'dest': 'search'}),
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
              'dest': 'json'})
        ]
    )
    def list(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        content_client = APIClient(db.get_configure()).get_content_api_client()
        enterprise_id = db.get_enterprise_id()

        search = self.app.pargs.search
        limit = self.app.pargs.limit
        offset = self.app.pargs.offset

        kwargs = {}
        if search:
            kwargs['search'] = search

        try:
            response = content_client.get_all_content(enterprise_id, limit=limit, offset=offset, **kwargs)
        except ApiException as e:
            self.app.log.error(f"[content-list] Failed to list contents: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        self.app.render(f"Total Number of Contents: {response.count}")
        if not self.app.pargs.json:
            contents = []

            label = {
                'id': "ID",
                'name': "NAME",
                'description': "DESCRIPTION",
                'tags': "TAGS",
                'size': "SIZE",
                'created_on': "CREATED ON"
            }

            for content in response.results:
                contents.append(
                    {
                        label['id']: content.id,
                        label['name']: content.name,
                        label['description']: content.description,
                        label['tags']: content.tags,
                        label['size']: content.size,
                        label['created_on']: content.created
                    }
                )
            self.app.render(contents, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            contents = []
            for content in response.results:
                contents.append(
                    {
                        'id': content.id,
                        'download_url': content.download_url,
                        'name': content.name,
                        'key': content.key,
                        'is_dir': content.is_dir,
                        'kind': content.kind,
                        'hash': content.hash,
                        'size': content.size,
                        'path': content.path,
                        'permissions': content.permissions,
                        'tags': content.tags,
                        'description': content.description,
                        'created': str(content.created),
                        'modified': str(content.modified),
                        'enterprise': content.enterprise,
                        'owner': str(content.owner)
                    }
                )
            self.app.render(contents, format=OutputFormat.JSON.value)


    def _content_basic_response(self, content, format=OutputFormat.TABULATED):
        enterprise_id = content.enterprise.split('/')[-2]
        if format == OutputFormat.TABULATED:
            title = "TITLE"
            details = "DETAILS"
            renderable = [
                {title: 'id', details: content.id},
                {title: 'name', details: content.name},
                {title: 'is_dir', details: content.is_dir},
                {title: 'kind', details: content.kind},
                {title: 'hash', details: content.hash},
                {title: 'size', details: content.size},
                {title: 'path', details: content.path},
                {title: 'permissions', details: content.permissions},
                {title: 'tags', details: content.tags},
                {title: 'description', details: content.description},
                {title: 'created', details: content.created},
                {title: 'modified', details: content.modified},
                {title: 'enterprise', details: enterprise_id},
                {title: 'owner', details: content.owner.username}
            ]
        else:
            renderable = {
                'id': content.id,
                'download_url': content.download_url,
                'name': content.name,
                'key': content.key,
                'is_dir': content.is_dir,
                'kind': content.kind,
                'hash': content.hash,
                'size': content.size,
                'path': content.path,
                'permissions': content.permissions,
                'tags': content.tags,
                'description': content.description,
                'created': str(content.created),
                'modified': str(content.modified),
                'enterprise': content.enterprise,
                'owner': str(content.owner)
            }

        return renderable


    
    @ex(
        help='Show content details',
        arguments=[
            (['content_id'],
             {'help': 'Content id',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        content_client = APIClient(db.get_configure()).get_content_api_client()
        enterprise_id = db.get_enterprise_id()

        content_id = self.app.pargs.content_id

        try:
            response = content_client.get_content(content_id, enterprise_id)
        except ApiException as e:
            self.app.log.error(f"[content-show] Failed to show details of the content: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._content_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._content_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)


    @ex(
        help='Upload content',
        arguments=[
            (['content_file'],
             {'help': 'File to upload',
              'action': 'store'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def upload(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        content_client = APIClient(db.get_configure()).get_content_api_client()
        enterprise_id = db.get_enterprise_id()

        content_file = self.app.pargs.content_file

        try:
            filesize = os.path.getsize(content_file)
            random_no = random.randint(1, 50)
            with tqdm(total=int(filesize), unit='B', unit_scale=True, miniters=1, desc='Uploading......',
                      unit_divisor=1024) as pbar:
                for i in range(100):
                    if i == random_no:
                        response = content_client.post_content(enterprise_id, content_file)

                    time.sleep(0.07)
                    pbar.set_postfix(file=Path(content_file).name, refresh=False)
                    pbar.update(int(filesize / 100))
        except ApiException as e:
            self.app.log.error(f"[content-upload] Failed to upload content: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._content_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._content_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)


    @ex(
        help='Modify content details',
        arguments=[
            (['content_id'],
             {'help': 'Content id',
              'action': 'store'}),
            (['-t', '--tags'],
             {'help': 'List of tags, space separated',
              'nargs': "*",
              'type': str,
              'dest': 'tags'}),
            (['-d', '--description'],
             {'help': 'Description',
              'action': 'store',
              'dest': 'description'}),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'})
        ]
    )
    def modify(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        content_client = APIClient(db.get_configure()).get_content_api_client()
        enterprise_id = db.get_enterprise_id()

        content_id = self.app.pargs.content_id
        tags = self.app.pargs.tags
        description = self.app.pargs.description

        data = {}
        if tags:
            data["tags"] = tags

        if description:
            data["description"] = description

        if not tags and not description:
            self.app.log.info('[content-modify] Both tags and description values are empty')
            self.app.render('Both tags and description values are empty\n')
            return

        try:
            response = content_client.patch_content(content_id, enterprise_id, data=data)
        except ApiException as e:
            self.app.log.error(f"[content-modify] Failed to modify content: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return

        if not self.app.pargs.json:
            renderable = self._content_basic_response(response)
            self.app.render(renderable, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            renderable = self._content_basic_response(response, OutputFormat.JSON)
            self.app.render(renderable, format=OutputFormat.JSON.value)

    @ex(
        help='Delete content',
        arguments=[
            (['content_id'],
             {'help': 'Content id',
              'action': 'store'})
        ]
    )
    def delete(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        content_client = APIClient(db.get_configure()).get_content_api_client()
        enterprise_id = db.get_enterprise_id()

        content_id = self.app.pargs.content_id

        try:
            content_client.delete_content(content_id, enterprise_id)
            self.app.log.debug(f"[content-delete] Content with id : {content_id} deleted successfully.")
            self.app.render(f"Content with id {content_id} deleted successfully.\n")

        except ApiException as e:
            self.app.log.error(f"[content-delete] Failed to delete the content: {e}")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)} \n")
            return