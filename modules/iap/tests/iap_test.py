import argparse
import json
import os
import requests
from google.auth.transport.requests import Request
from google.oauth2 import id_token
from googleapiclient import discovery
import google.auth
from google.cloud import resourcemanager_v3

def get_project_number(project_id):
  # Create a Resource Manager client
  client = resourcemanager_v3.ProjectsClient()

  # Initialize request argument(s)
  request = resourcemanager_v3.GetProjectRequest(
      name=f'projects/{project_id}',
  )

  # Make the request
  response = client.get_project(request=request)

  # Handle the response
  print(response)
  return response.name.split('/')[1]

def get_project_id():
  id = ""
  if 'GCP_PROJECT' in os.environ:
    id = os.environ['GCP_PROJECT']
  elif 'GOOGLE_APPLICATION_CREDENTIALS' in os.environ:
    with open(os.environ['GOOGLE_APPLICATION_CREDENTIALS'], 'r') as fp:
      credentials = json.load(fp)
    id = credentials['project_id']
  return id

def list_backend_services_ids(project_id, keyword):
  credentials, _ = google.auth.default()
  service = discovery.build('compute', 'v1', credentials=credentials)
  print(f'project id is {project_id}')
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

  # project_id = get_project_id()
  gcp_backend_services_ids = list_backend_services_ids(project_id, f'{namespace}-{keyword}')
  print("GCP Backend Services IDs:", gcp_backend_services_ids)
  project_number = get_project_number(project_id)

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
    print(resp.text)
    if resp.status_code == 403:
      raise Exception("Service account does not have permission to access the IAP-protected application.")
    elif resp.status_code != 200:
      raise Exception("Bad response from application: {!r} / {!r} / {!r}".format(resp.status_code, resp.headers, resp.text))
    else:
      return resp.text
  except requests.exceptions.RequestException as e:
    print(e)

# Define the argument parser and add arguments
parser = argparse.ArgumentParser(description="Script to test GCP backend services via IAP.")
parser.add_argument('frontend_url', help="URL of the frontend service")
parser.add_argument('frontend_client_id', help="Client ID for the frontend service")
parser.add_argument('jupyter_url', help="URL of the Jupyter service")
parser.add_argument('jupyter_client_id', help="Client ID for the Jupyter service")
parser.add_argument('ray_dashboard_url', help="URL of the Ray dashboard")
parser.add_argument('ray_dashboard_client_id', help="Client ID for the Ray dashboard")
parser.add_argument('project_id', help="GCP Project ID")
parser.add_argument('namespace', help="Namespace for the backend services")

# Parse the arguments
args = parser.parse_args()

# Use the parsed arguments in the rest of your script
project_id = args.project_id
namespace = args.namespace

def test_jupyter():
  r = make_iap_request(args.jupyter_url, args.jupyter_client_id, "jupyter")
  print(r)

def test_frontend():
  r = make_iap_request(args.frontend_url, args.frontend_client_id, "frontend")
  print(r)

def test_ray_dashboard():
  r = make_iap_request(args.ray_dashboard_url, args.ray_dashboard_client_id, "ray-dashboard")
  print(r)

test_jupyter()
test_frontend()
test_ray_dashboard()
