/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

output "created_resources" {
  description = "IDs of the resources created, if any."
  value = merge(
    var.project_create == null ? {} : {
      project = module.project.project_id
    },
    !local.vpc_create ? {} : {
      subnet_id = one(values(module.vpc.0.subnet_ids))
      vpc_id    = module.vpc.0.id
    },
    !var.registry_create ? {} : {
      registry = module.registry.0.image_path
    },
    !local.cluster_create ? {} : (
      var.gke_autopilot ?
      {
        cluster              = module.cluster-autopilot.0.id
        node_service_account = module.cluster-service-account.0.email
      }
      :
      {
        cluster              = module.cluster-standard.0.id
        node_service_account = module.cluster-service-account.0.email
      }
    ),
    !local.create_nat ? {} : {
      router    = module.nat.0.id
      cloud_nat = module.nat.0.router.id
    },
    local.proxy_only_subnet == null ? {} : {
      proxy_only_subnet = one(values(module.vpc.0.subnets_proxy_only)).id
    }
  )
}

output "project_id" {
  description = "Project ID of where the GKE cluster is hosted"
  value       = var.project_create == null ? var.project_id : module.project.project_id
}

output "fleet_host" {
  description = "Fleet Connect Gateway host that can be used to configure the GKE provider."
  value = join("", [
    "https://connectgateway.googleapis.com/v1/",
    "projects/${local.fleet_project.number}/",
    "locations/global/gkeMemberships/${var.cluster_name}"
  ])
}

output "get_credentials" {
  description = "Run one of these commands to get cluster credentials. Credentials via fleet allow reaching private clusters without no direct connectivity."
  value = {
    direct = join("", [
      "gcloud container clusters get-credentials ${var.cluster_name} ",
      "--project ${var.project_id} --location ${var.gke_location}"
    ])
    fleet = join("", [
      "gcloud container fleet memberships get-credentials ${var.cluster_name}",
      " --project ${var.project_id}"
    ])
  }
}
