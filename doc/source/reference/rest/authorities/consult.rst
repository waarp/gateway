Consulter une autorité
======================

.. http:get:: /api/authorities/(string:authority_name)

   Renvoie l'autorité demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: L'autorité a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'autorité demandée n'existe pas

   :resjson string name: Le nom de l'autorité
   :resjson string type: Le type d'autorité (TLS, SSH...)
   :resjson string publicIdentity: La valeur d'identité publique (certificat,
      clé publique...) de l'autorité
   :resjson array validHosts: La liste des hôtes que l'autorité est habilitée à
      authentifier. Si vide, l'autorité peut authentifier tous les hôtes.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/authorities/tls_ca HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 979

         {
           "name": "tls_ca",
           "type": "tls_authority",
           "publicIdentity": "-----BEGIN CERTIFICATE-----
             MIICMzCCAZygAwIBAgIRAJFIx3lh/L57UPaTaMcBJ8wwDQYJKoZIhvcNAQELBQAw
             EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
             MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
             gYkCgYEAyLZU8wra4/hvLmEJWD/mdCq3BVW2zqmEa7gYZKyrNSN+iOzu9sLUR3fx
             oo5UYT87x6xi+762QI+yiwZOxkdkbKv2yQXqpF6CO1J2IuCjbdwV9ZLapGsLT2jt
             RUyR2w8qSQP7pl1Lk1K8mos+sdsRINX4VmsLG/pOukMyvUu7NTECAwEAAaOBhjCB
             gzAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/
             BAUwAwEB/zAdBgNVHQ4EFgQUMlkJ+EgiVFx6OlaVub4NQ9HgRwEwLAYDVR0RBCUw
             I4IJbG9jYWxob3N0hwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMA0GCSqGSIb3DQEB
             CwUAA4GBAGnpw8im001qnW+e+V339MBTabqvXvsaMKIf75+sYkGsFhLOYw+kT4fg
             31bd3B7u5azc/FKfQdDOjjhvnGqoHtyjjVMhxLIN0fjugMTGxw4Er5xIC5RGuynB
             lqNcbCum94NGVmx0wDs3WOgcN0GCpiasPZcFs7VoVanerLOBIMXj
             -----END CERTIFICATE-----",
           "validHosts": ["1.2.3.4", "waarp.org"]
         }