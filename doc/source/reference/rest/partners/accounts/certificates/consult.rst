[OBSOLÈTE] Consulter un certificat
==================================

.. http:get:: /api/partners/(string:partner)/accounts/(string:login)/certificates/(string:cert_name)

   Renvoie le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le certificat a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou le certificat demandés n'existent pas

   :resjson string name: Le nom du certificat
   :resjson string privateKey: La clé privée du certificat en format PEM
   :resjson string certificate: Le certificat de l'entité en format PEM (mutuellement
      exclusif avec `public_key`)
   :resjson string publicKey: La clé publique SSH de l'entité en format *authorized_key*
      (mutuellement exclusif avec `certificate`)


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/partners/waarp_r66/accounts/titi/certificates/certificat_titi HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 685

         {
           "name": "certificat_titi",
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
             MIICJTCCAY6gAwIBAgIQIKHvcsM3cly5gnpNEdxSXTANBgkqhkiG9w0BAQsFADAS
             MRAwDgYDVQQKEwdBY21lIENvMCAXDTgwMDEwMTAwMDAwMFoYDzIwOTQwMTI4MTYw
             MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
             iQKBgQDPaGdQDzA/C4GaCgouUUXn0ngCIyNTTn6bjUSYI2Vd8EhILreb2Bl6848A
             ScAR6E4+vlvNo7rWAGP9pHS2JqCfio/LHcudFoEiFvgEfk/+2WL9JNXjlRSBsuZm
             tXlKLgb4Zg6NCQrFLH3HmJbo/EWRp716aXxfz6gbJYXg62a5GQIDAQABo3oweDAO
             BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwIwDwYDVR0TAQH/BAUw
             AwEB/zAdBgNVHQ4EFgQUnjMDLHQqjzwodoDHRu82q9zkLLwwIQYDVR0RBBowGIcE
             fwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOBgQAD0G1ENOLg
             Wf06w9SikUMzDHXWUsVA8PrODpWU0cmDY06sdpa4IIWKmhf95BVXnrOjJy7d3y1N
             b1Wte/HVOk8zgAta5W5WnQAMPvXuXFaC3Jy0YmQfY1rSjl/PLbXzA0gO0IcP93UF
             hZ0if1CWX+PzVETBXFKURT905E5qS+Ebng==
             -----END CERTIFICATE-----"
         }