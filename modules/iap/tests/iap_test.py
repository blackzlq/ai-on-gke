import sys
import requests
from google.auth.transport.requests import Request
from google.oauth2 import id_token

frontend_url = sys.argv[1]
frontend_client_id = sys.argv[2]
jupyter_url = sys.argv[3]
jupyter_client_id = sys.argv[4]
ray_dashboard_url = sys.argv[5]
ray_dashboard_client_id = sys.argv[6]
def make_iap_request(url, client_id, method="GET", **kwargs):
  if "timeout" not in kwargs:
    kwargs["timeout"] = 90

  open_id_connect_token = id_token.fetch_id_token(Request(), client_id)
  print(open_id_connect_token)
  resp = requests.request(
      method,
      url,
      headers={"Authorization": "Bearer {}".format(open_id_connect_token)},
      **kwargs
  )
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

def test_jupyter():
  r = make_iap_request(jupyter_url, jupyter_client_id)
  print(r.content.decode('utf-8'))

def test_frontend():
  r = make_iap_request(frontend_url, frontend_client_id)
  print(r.content.decode('utf-8'))

def test_ray_dashboard():
  r = make_iap_request(ray_dashboard_url, ray_dashboard_client_id)
  print(r.content.decode('utf-8'))

test_jupyter()
test_frontend()
test_ray_dashboard()