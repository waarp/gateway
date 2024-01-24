Ajouter une valeur d'authentification
=====================================

.. http:post:: /api/servers/(string:server_name)/credentials

   Ajoute une nouvelle valeur d'authentification au serveur donné.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de la valeur d'authentification. Par défaut, le
     nom du type est utilisé.
   :reqjson string type: Le type d'authentification utilisé. Voir les
     :ref:`reference-auth-methods` pour la liste des différents type d'authentification
     supportés.
   :reqjson string value: La valeur primaire d'authentification.
   :reqjson string value2: La valeur secondaire d'authentification (dépend du
     type d'authentification).

   :statuscode 201: La valeur d'authentification a été créée avec succès.
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides.
   :statuscode 401: Authentification d'utilisateur invalide.

   :resheader Location: Le chemin d'accès au nouveau serveur créé.


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/servers/gw_r66/credentials HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 2410

         {
           "name": "r66_certificate",
           "type": "tls_certificate",
           "value": "-----BEGIN CERTIFICATE-----
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
           "value2": "-----BEGIN PRIVATE KEY-----
             MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBAMi2VPMK2uP4by5h
             CVg/5nQqtwVVts6phGu4GGSsqzUjfojs7vbC1Ed38aKOVGE/O8esYvu+tkCPsosG
             TsZHZGyr9skF6qRegjtSdiLgo23cFfWS2qRrC09o7UVMkdsPKkkD+6ZdS5NSvJqL
             PrHbESDV+FZrCxv6TrpDMr1LuzUxAgMBAAECgYEAmKww5Arew8gG8lV3oTxCFR0k
             yJcRjhPeGX4YeAPr22jbaFYp02QRyydOk2MGhk5uL41OYcYIpgVoP14V77cAiFvJ
             V4GMSvp4+YACzK2/kpCm6vdeZJlrbix32kHJo3+HgCa7tlW5NyLjcRuGBdECsqqv
             pDHb/Gb7oppXJqXjHQ0CQQDWeUJYVgwS+aTAFsdeDmQWwSDRksiBo6Y55LJAPR+U
             zaHuQ4MeWm1olPwPwHsAlIJIvdlF31i9j7kaNC14/smvAkEA75L259DZj5JIpMP+
             3x0pxduam0Mf258ibxNiuL/T3qt4kBwTUa2JlIFVNPlwc2sqhc0eJliTvGOWhpOG
             2LMHHwJBAIH2jtZ6pexlrIjeBMehDtOfCiUUvj2YjiTsyXsVzupbxTFdZbnh8AR8
             q1VcPOz4EQ7FREEL+3k6+16+mYOFWW8CQBtdkDJ+mrtZnE6lzLEzpZfiM9DUZAk0
             Ljy93CL6Vnsy3vynGFXWGscJ1u/MJloovZy3B2Cd8ZItVf5dT6PlH0UCQQCxlbQB
             uxd0sOxCNiQOnnD12ma8nn21Vex34lvKIAI6Zi55ikCtrezcrB67DI9ANEq8w6vl
             sOYLXpHzyo7oGMVz
             -----END PRIVATE KEY-----"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/gw_r66/authentication/r66_certificate
