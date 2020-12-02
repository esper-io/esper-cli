import requests

class APIException(Exception):
    pass


def get_collection_api_url():
    return 'http://localhost:8000/api/v0/enterprise/796e7ab9-ad58-49ea-bc32-4fb9241b2a30/collection/'

def get_eql_search_url():
    return 'http://localhost:8000/api/v0/enterprise/796e7ab9-ad58-49ea-bc32-4fb9241b2a30/collection/search/'

def eql_search(eql_query):
    url = get_eql_search_url()
    query_params = {"q" : eql_query}
    try:
        response = requests.get(
            url,
            auth=('siteadmin','siteadmin123$'),
            params=query_params
        )
    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


