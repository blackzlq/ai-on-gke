import sys
import requests
from google.auth.transport.requests import Request
from google.oauth2 import id_token
from googleapiclient import discovery
import google.auth
from google.cloud import resource_manager

def get_project_number(project_id):
  # Create a Resource Manager client
  client = resource_manager.Client()

  # Fetch the project
  project = client.fetch_project(project_id)

  # Return the project number
  return project.number



frontend_url = sys.argv[1]
frontend_client_id = sys.argv[2]
jupyter_url = sys.argv[3]
jupyter_client_id = sys.argv[4]
ray_dashboard_url = sys.argv[5]
ray_dashboard_client_id = sys.argv[6]
project_id = sys.argv[7]
namespace = sys.argv[8]

def list_backend_services_ids(project_id, keyword):
  credentials, _ = google.auth.default()
  service = discovery.build('compute', 'v1', credentials=credentials)
  request = service.backendServices().list(project=project_id)
  response = request.execute()

  filtered_service_ids = [
      service['id'] for service in response.get('items', [])
      if keyword.lower() in service['name'].lower()
  ]

  return filtered_service_ids

def make_iap_request(url, client_id, keyword, method="GET", **kwargs):
  if "timeout" not in kwargs:
    kwargs["timeout"] = 90

  # List GCP backend services IDs based on the project ID and keyword
  gcp_backend_services_ids = list_backend_services_ids(project_id, f'{namespace}-{keyword}')
  print("GCP Backend Services IDs:", gcp_backend_services_ids)
  project_number = get_project_number(project_id)

  # Construct expected audiences from the GCP backend services IDs
  expected_audiences = [f"/projects/{project_number}/global/backendServices/{service_id}" for service_id in gcp_backend_services_ids]
  print("Expected Audiences:", expected_audiences)

  open_id_connect_token = id_token.fetch_id_token(Request(), expected_audiences[0])
  print(f'token is {open_id_connect_token}')
  print(f'url is {url}')
  try:
    resp = requests.request(
        method,
        url,
        headers={"Authorization": "Bearer {}".format(open_id_connect_token)},
        **kwargs
    )
    # If the request was successful, you can now process the response
    print(resp.text)
    if resp.status_code == 403:
      raise Exception(
          "Service account does not have permission to "
          "access the IAP-protected application."
      )
    elif resp.status_code != 200:
      raise Exception(
          "Bad response from application: {!r} / {!r} / {!r}".format(
              resp.status_code, resp.headers, resp.text
          )
      )
    else:
      return resp.text
  except requests.exceptions.RequestException as e:
    # This will catch any errors during the request
    print(e)



def test_jupyter():
  r = make_iap_request(jupyter_url, jupyter_client_id, "jupyter")
  print(r.content.decode('utf-8'))

def test_frontend():
  r = make_iap_request(frontend_url, frontend_client_id, "frontend")
  print(r.content.decode('utf-8'))

def test_ray_dashboard():
  r = make_iap_request(ray_dashboard_url, ray_dashboard_client_id, "ray-dashboard")
  print(r.content.decode('utf-8'))

test_jupyter()
test_frontend()
test_ray_dashboard()