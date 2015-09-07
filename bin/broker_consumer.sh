http GET 127.0.0.1:8001/v2/catalog

http PUT 127.0.0.1:8001/v2/service_instances/11111111-11111111-11111111-11111111 service_id=1 plan_id=1 organization_guid=1 space_guid=1 parameters:='{}'

http GET 127.0.0.1:8001/v2/service_instances/11111111-11111111-11111111-11111111/last_operation

http PUT 127.0.0.1:8001/v2/service_instances/11111111-11111111-11111111-11111111/service_bindings/11111111-11111111-11111111-11111111 service_id=1 plan_id=1 app_guid=1

http DELETE 127.0.0.1:8001/v2/service_instances/11111111-11111111-11111111-11111111/service_bindings/11111111-11111111-11111111-11111111

http DELETE 127.0.0.1:8001/v2/service_instances/11111111-11111111-11111111-11111111
