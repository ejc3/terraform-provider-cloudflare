output "access_application_worker_destinations" {
  value = {
    for application in data.cloudflare_zero_trust_access_applications.example_zero_trust_access_applications.result :
    application.id => application.destinations
  }
}
