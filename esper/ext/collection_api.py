import requests

class APIException(Exception):
    pass


def get_collection_url(environment: str,enterprise_id: str, collection_id: str = None) -> str:
    """
    Build and return collection url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param collection_id:
    :return: Url
    """

    url = f'https://{environment}-api.esper.cloud/api/v0/enterprise/{enterprise_id}/collection/'

    if collection_id:
        url += f'{collection_id}/'

    return url


def get_eql_search_url(environment: str,enterprise_id: str) -> str:
    """
    Build and return collection search url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param collection_id:
    :return: Url
    """
    url = get_collection_url(environment,enterprise_id) + f'search/'
    return url


def eql_search(environment: str, enterprise_id: str, api_key: str, eql_query: str) -> str:
    url = get_eql_search_url(environment,enterprise_id)
    query_params = {"q" : eql_query}
    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            params=query_params
        )
    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


