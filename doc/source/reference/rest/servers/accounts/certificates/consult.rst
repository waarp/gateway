Consulter un certificat
=======================

.. http:get:: /api/servers/(string:server)/accounts/(string:login)/certificates/(string:cert_name)

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

         GET https://my_waarp_gateway.net/api/servers/gw_r66/accounts/toto/certificates/certificat_toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 312

         {
           "name": "certificat_toto",
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