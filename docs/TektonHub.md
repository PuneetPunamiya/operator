# Tekton Hub

TektonHub custom resource allows user to install and manage [Tekton Hub][hub].

TektonHub is an optional component and currently cannot be installed through TektonConfig. It has to be installed seperately. It is available for both Kubernetes and OpenShift platform.

To install Tekton Hub on your cluster follow steps as given below:

1.  Make sure that `tekton-pipelines` namespace is present on your
    Kubernetes cluster or `openshift-pipelines` namespace in case of
    OpenShift cluster as installation is going to happen in this namespace.
2.  Create the secrets for `postgres` database.
    A database root password must be generated and stored in a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/)
    before installing Tekton Hub. By default, Tekton Hub expects this secret to have
    the following properties:

    - namespace: `tekton-pipelines`(in case of Kubernetes) /`openshift-pipelines`(in case of OpenShift)
    - name: `tekton-hub-db`
    - contains the fields:

      - `POSTGRES_DB=<dbname>`
      - `POSTGRES_USER=<postgresuser>`
      - `POSTGRES_PASSWORD=<postgrespassword>`
      - `POSTGRES_PORT="5432"`

    > Note: If the secret for the DB is not created then by default operator is
    > going to create one for us.

3.  Similarly create the secrets for the API before we install Tekton Hub. By default,
    Tekton Hub expects this secret to have the following properties:

    - namespace: `tekton-pipelines`(in case of Kubernetes) /`openshift-pipelines`(in case of OpenShift)
    - name: `tekton-hub-api`
    - contains the fields:

      - `GH_CLIENT_ID=<github-client-id>`
      - `GH_CLIENT_SECRET=<github-client-secret>`
      - `JWT_SIGNING_KEY=<jwt-signing-key>`
      - `ACCESS_JWT_EXPIRES_IN=<time(eg 30d)>`
      - `REFRESH_JWT_EXPIRES_IN=<time(eg 30d)>`
      - `GHE_URL=<github enterprise url(leave it blank if not using github enterprise>`

    > _Note 1_: In order to get more details on how to get Github Client ID and Secret
    > you can find the steps [here](https://docs.github.com/en/developers/apps/building-oauth-apps/creating-an-oauth-app).
    > _Note 2_: If the fields for API secret are left blank except for
    > `GHE_URL` then the operator will go into error state.

4.  Once the secrets are created now we need to understand how TektonHub CR looks.

    ```yaml
    apiVersion: operator.tekton.dev/v1alpha1
    kind: TektonHub
    metadata:
      name: hub
    spec:
      db:
        secret: db
      api:
        secret: api
        hubConfigUrl: https://raw.githubusercontent.com/tektoncd/hub/main/config.yaml
        # kubernetes specific fields not required in case of OpenShift
        ingressHostUrl: api.host
        ingressClassName: ambassador
    ```

    ### DB

    The following field helps to configure the DB deployment. Provided fields are:

    - `secret`: Name of the DB secret created in `tekton-pipelines`/`openshift-pipelines` namespace.

    ### API

    The following field helps to configure the API deployment. Provided fields are:

    - `secret`: Name of the API secret created in `tekton-pipelines`/`openshift-pipelines` namespace.
    - `hubConfigUrl`: The place of Tekton Hub config url as shown above.

      #### Kubernetes specific fields:

      - `ingressHostUrl`: This is required in case of Kubernetes, the Host URL to expose API.
      - `ingressClassName`: The ingress class being used in case of Kubernetes.

5.  After configuring the TektonHub spec you can install Tekton Hub by running the command

    ```sh
    kubectl apply -f <name>.yaml
    ```

6.  Check the status of installation using following command

    ```sh
    kubectl get tektonhub.operator.tekton.dev
    ```

[hub]: https://github.com/tektoncd/hub
