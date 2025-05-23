[OBSOLÈTE] Consulter un certificat
==================================

.. http:get:: /api/partners/(string:partner)/certificates/(string:cert_name)

   Renvoie le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le certificat a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou le certificat demandés n'existent pas

   :resjson string name: Le nom du certificat
   :resjson string privateKey: La clé privée du certificat en format PEM
   :resjson string certificate: Le certificat de l'entité en format PEM (mutuellement
      exclusif avec ``public_key``)
   :resjson string publicKey: La clé publique SSH de l'entité en format
      ``authorized_key`` (mutuellement exclusif avec ``certificate``)


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/partners/waarp_r66/certificates/certificat_waarp HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 685

      {
        "name": "certificat_waarp",
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
