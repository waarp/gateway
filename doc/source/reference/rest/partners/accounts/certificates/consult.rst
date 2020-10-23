Consulter un certificat
=======================

.. http:get:: /api/partners/(string:partner)/accounts/(string:login)/certificates/(string:cert_name)

   Renvoie le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur


   :statuscode 200: Le certificat a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire, le compte ou le certificat demandés n'existent pas

   :resjson string name: Le nom du certificat
   :resjson string privateKey: La clé privée du certificat
   :resjson string publicKey: La clé publique du certificat
   :resjson string certificate: Le certificat de l'entité


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/partners/waarp_sftp/accounts/titi/certificates/certificat_titi HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 197
      
      {
        "name": "certificat_titi",
        "privateKey": "<clé privée>",
        "publicKey": "<clé publique>",
        "cert": "<certificat>"
      }
