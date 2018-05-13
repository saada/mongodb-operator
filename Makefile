all: build redeploy
build:
	operator-sdk build saada/mongodb-operator
	docker push saada/mongodb-operator
redeploy:
	# delete resources
	kubectl delete -f deploy/cr.yaml
	kubectl delete -f deploy/operator.yaml
	kubectl delete -f deploy/crd.yaml
	kubectl delete -f deploy/rbac.yaml

	# create resources
	kubectl create -f deploy/rbac.yaml
	kubectl create -f deploy/crd.yaml
	kubectl create -f deploy/operator.yaml
	kubectl create -f deploy/cr.yaml
regenerate:
	operator-sdk generate k8s
