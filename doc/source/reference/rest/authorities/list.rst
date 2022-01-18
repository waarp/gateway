Lister les autorités
====================

.. http:get:: /api/authorities

   Renvoie une liste des autorités remplissant les critères donnés en
   paramètres de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les autorités seront triées *(défaut: name+)*
   :type sort: [name+|name-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array authorities: La liste des autorités demandées
   :resjsonarr string name: Le nom de l'autorité
   :resjsonarr string type: Le type d'autorité (TLS, SSH...)
   :resjsonarr string publicIdentity: La valeur d'identité publique (certificat,
      clé publique...) de l'autorité
   :resjsonarr array validHosts: La liste des hôtes que l'autorité est habilitée
      à authentifier. Si vide, l'autorité peut authentifier tous les hôtes.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/authorities?limit=10&sort=name+ HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 1862

         {
           "authorities": [{
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
           },{
             "name": "ssh_ca",
             "type": "ssh_cert_authority",
             "publicIdentity": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhbxVecyg3Nb
               OuGgIbzUuB3GyVIKBRWhYUOaEJtqMR8ckb3WM6cy0yplbZ6is4y8gqGhE9pQ8g3JrbY
               Ylrb8/HnjuCnSzA9BVhMNUxp/9Ar7GSvBO2bPIcPYBePe19AJ6MsjoT2jcZhUwlsiac
               HAnRWaOfYeJQP0Fw9zqhhPcjOnWIewNQaghwBXyyzQB/BbiYAMvPo0uveYY+Yr18ExI
               v3ybtqgAgSVHji4Jg4JFwVd9VPfAz3y4ucEYiOr/4bkOBTuAMxbvE+S8mvbOTQ+itsF
               QxuJgWTrx/53Yth3QYDwgjTaT7TLSSRpi1+s9QQg6XTanJyjtEmmYbnaB+EhAQfI0mf
               OripP/1cTq9StZfYTKl58ObrYWmc5CDH338uCdK5GxIP9eNz4RcLqPLvcVBrm62qsYR
               eoD62InykggeOSgkOo4UGbC7JSEdW3afMBGdh797eht6qX3ywKbs7GNVwOt2M7xrpmC
               ehU1uegN7GtIRvCZR0JH4+KSGitWFY3E=",
             "validHosts": ["9.8.7.6", "waarp.fr"]
           }]
         }