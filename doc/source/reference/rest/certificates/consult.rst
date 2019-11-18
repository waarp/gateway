Consulter un certificat
=======================

.. http:get:: /api/certificates/(int:certificate_id)

   Renvoie le certificat portant le numéro ``certificate_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/certificates/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le certificat a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat demandé n'existe pas

   :resjson number id: L'identifiant unique du certificat
   :resjson string name: Le nom du certificat
   :resjson string ownerType: Le type d'entité
   :resjson number ownerID: L'identifiant de l'entité à laquelle appartient le certificat
   :resjson string privateKey: La clé privée de l'entité
   :resjson string publicKey: La clé publique de l'entité
   :resjson string cert: Le certificat de l'entité

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 197

          {
            "id": 1,
            "name": "certificat_sftp",
            "ownerType": "local_agents",
            "ownerID": 1,
            "privateKey": "<clé privée>",
            "publicKey": "<clé publique>",
            "cert": "<certificat>"
          }