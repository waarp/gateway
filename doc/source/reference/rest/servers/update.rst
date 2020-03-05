Modifier un serveur
===================

.. http:put:: /api/servers/(int:server_id)

   Met à jour le serveur portant le numéro ``server_id`` avec les informations
   renseignées en format JSON dans le corps de la requête. Les champs non-spécifiés
   resteront inchangés.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON.

   **Exemple de requête**

       .. code-block:: http

          PATCH https://my_waarp_gateway.net/api/servers/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 83

          {
            "name": "sftp_server_new",
            "protocol": "sftp",
            "protoConfig": {
              "address": "localhost",
              "port": 23,
              "root": "/new/sftp/root"
            }
          }


   **Réponse**

   :statuscode 201: Le serveur a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas

   :resheader Location: Le chemin d'accès au serveur modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/servers/1