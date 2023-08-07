[OBSOLÈTE] Lister les certificats
=================================

.. http:get:: /api/servers/(string:server)/certificates

   Renvoie une liste des certificats emplissant les critères donnés en paramètres
   de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les certificats seront triés *(défaut: name+)*
   :type sort: [name+|name-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resjson array certificates: La liste des certificats demandés
   :resjsonarr string name: Le nom du certificat
   :resjsonarr string privateKey: La clé privée du certificat en format PEM
   :resjsonarr string certificate: Le certificat de l'entité en format PEM
      (mutuellement exclusif avec ``public_key``)
   :resjsonarr string publicKey: La clé publique SSH de l'entité en format
      ``authorized_key`` (mutuellement exclusif avec ``certificate``)


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/servers/serveur_r66/certificates?limit=5 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 453

      {
        "certificates": [{
          "name": "certificat_r66",
          "privateKey": "-----BEGIN PRIVATE KEY-----
            MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAM9oZ1APMD8LgZoK
            Ci5RRefSeAIjI1NOfpuNRJgjZV3wSEgut5vYGXrzjwBJwBHoTj6+W82jutYAY/2k
            dLYmoJ+Kj8sdy50WgSIW+AR+T/7ZYv0k1eOVFIGy5ma1eUouBvhmDo0JCsUsfceY
            luj8RZGnvXppfF/PqBslheDrZrkZAgMBAAECgYEAjHHsE4BVcTt/ZSmLP1X1ekdA
            0GGu2Ah9HyQH4OWHDJdautY3qqYoiuNGYDGQiA/AfCg2zgciyyq0itrD1VxOwsG0
            dO7yu5i9ooWnETV/tTZq1aM4HyeXaK/dl1LzJ+tBIVOeGa3AMQvSF84IjJEN9dYg
            2a4BUh/nt+fmRNb52SECQQDupRSvff1rTmBjrZOOs9s56GSMryyjvggJHYcBhSyk
            liagybxWxCinkUP0VdfESzd9j6xDhygO2Islq0BFr9FtAkEA3n3GNKpAzQ2QlyRr
            w5cMECypYXdPyjNAG6rP/HB4adWJRxnMAGglRSmYNjitHLxG0+wo0IfDXq/5f4wZ
            yvPm3QJAZhBqWWf8A3HA3cC11BluEEUpA9ZDtEAo9aUQQYEwh6/EE45UI5O/g3Mo
            ag5wun4k3GmfFj5uznKkiFbGpUc9vQJAKvBLGE7jQq+jgAffZFf6VATKi6zjETri
            3HQSv71U/9feLoKkBFAVIUvtvEkj36/WW3/wQI5y/gsoM51uPOTlYQJAVhFbI4s2
            Zht/QWMq1v8BtVVZIFRksEIn3LIHga7Q5HpkqXmpl9lNh7s0DAvReDb3wyW0UxJS
            vkxL195flB04sw==
            -----END PRIVATE KEY-----",
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
        }]
      }
