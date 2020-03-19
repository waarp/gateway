Consulter un serveur
====================

.. http:get:: /api/servers/(int:server_id)

   Renvoie les informations du serveur portant l'identifiant ``server_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/servers/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Les informations du serveur ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas

   :resjson number id: L'identifiant unique du serveur
   :resjson string name: Le nom du serveur
   :resjson string protocol: Le protocole utilisé par le serveur
   :resjson string root: Le dossier racine du serveur
   :resjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON.

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 118

          {
            "id": 1,
            "name": "sftp server",
            "protocol": "sftp",
            "root": "/sftp/root",
            "protoConfig": {
              "address": "localhost",
              "port": 21
            }
          }