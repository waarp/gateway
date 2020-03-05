Ajouter un certificat
=====================

.. http:post:: /api/certificates

   Ajoute un nouveau certificat avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string ownerType: Le type d'entité
   :reqjson number ownerID: L'identifiant de l'entité à laquelle appartient le certificat
   :reqjson string privateKey: La clé privée de l'entité
   :reqjson string publicKey: La clé publique de l'entité
   :reqjson string certificate: Le certificat de l'entité

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/certificates HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 156

          {
            "name": "certificat_sftp",
            "ownerType": "local_agents",
            "ownerID": 1,
            "privateKey": "<clé privée>",
            "publicKey": "<clé publique>",
            "cert": "<certificat>"
          }

   **Réponse**

   :statuscode 201: Le certificat a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du certificat sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau certificat créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/certificates/1
