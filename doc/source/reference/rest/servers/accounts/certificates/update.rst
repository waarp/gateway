[OBSOLÈTE] Modifier un certificat
=================================

.. http:patch:: /api/servers/(string:server)/accounts/(string:login)/certificates/(string:cert_name)

   Met à jour le certificat demandé à partir des informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string privateKey: La clé privée du certificat en format PEM
   :reqjson string certificate: Le certificat de l'entité en format PEM (mutuellement
      exclusif avec ``public_key``)
   :reqjson string publicKey: La clé publique SSH de l'entité en format ``authorized_key``
      (mutuellement exclusif avec ``certificate``)

   :statuscode 201: Le certificat a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou le certificat demandés n'existent pas

   :resheader Location: Le chemin d'accès au certificat modifié


   **Exemple de requête**

   .. code-block:: http

      PATCH https://my_waarp_gateway.net/api/servers/gw_r66/accounts/toto/certificates/certificat_toto HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 665

      {
        "name": "certificat_toto_new",
        "certificate": "-----BEGIN CERTIFICATE-----
          MIICPDCCAaWgAwIBAgIRAMozibNPf0LHnyUC25vjrzQwDQYJKoZIhvcNAQELBQAw
          EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
          MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
          gYkCgYEAzAWD0DQX+nwfZcM3ZRnAAjAxCBM5SOsmMsr9rrgdXkZVrJ+e2obw3wYU
          kWNtmzCE4oKLgkXz7amrc4Z5MfJ/UROGURDge/PwWRa6PgCyHQK2TA2vup1GH16n
          +2uE7gOtCPHzENGIsN2bqHx9suO+NsO2+56A/AulQfNLYYEszbcCAwEAAaOBjzCB
          jDAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/
          BAUwAwEB/zAdBgNVHQ4EFgQU3Dn86/SOlQoDldWdm3831wOsGKwwNQYDVR0RBC4w
          LIIOMTI3LjAuMC4xOjY2NjaCCls6OjFdOjY2NjaCDmxvY2FsaG9zdDo2NjY2MA0G
          CSqGSIb3DQEBCwUAA4GBAFFL4e0IBbdxK8ohjnZz5c5PuCXzQy14fqVCozcHGVaf
          SKpWXKwjJnCpAmgzgwz60wFQuXAZNMxhCSTOxsuHrgJb+8EBNwiB8L1QNvI0TwQj
          7a9xLI4RZOju8VUANmTztJajWV+29Hs4fJkHKZtPvMhOAt0SWp1D9lxB6ChxY5c3
          -----END CERTIFICATE-----"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/servers/gw_r66/accounts/toto/certificates/certificat_toto
