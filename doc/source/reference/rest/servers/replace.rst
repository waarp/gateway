Remplacer un serveur
====================

.. http:put:: /api/servers/(int:server_id)

   Remplace le serveur portant le numéro ``server_id`` par celui renseigné
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront
   remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson string protoConfig: La configuration du serveur encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de requête**

       .. code-block:: http

          PUT https://my_waarp_gateway.net/api/servers/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 83

          {
            "name": "sftp_server_new",
            "protocol": "sftp",
            "protoConfig": "{\"address\":\"localhost\",\"port\":22}
          }


   **Réponse**

   :statuscode 201: Le serveur a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas

   :resheader Location: Le chemin d'accès au serveur modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/servers/1