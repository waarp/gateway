Créer un serveur
===================

.. http:post:: /api/servers

   Ajoute un nouveau serveur avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du serveur
   :reqjson string protocol: Le protocole utilisé par le serveur
   :reqjson string protoConfig: La configuration du serveur encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/servers HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 88

          {
            "name": "sftp_server",
            "protocol": "sftp",
            "protoConfig": "{\"address\":\"localhost\",\"port\":21}
          }


   **Réponse**

   :statuscode 201: Le serveur a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du serveur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau serveur créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/servers/1
