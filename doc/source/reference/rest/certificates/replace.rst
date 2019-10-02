Remplacer un certificat
=======================

.. http:put:: /api/certificates/(int:certificate_id)

   Remplace le certificat portant le numéro ``certificate_id`` par celui renseigné
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront
   remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string ownerType: Le type d'entité
   :reqjson number ownerID: L'identifiant de l'entité à laquelle appartient le certificat
   :reqjson string privateKey: La clé privée de l'entité
   :reqjson string publicKey: La clé publique de l'entité
   :reqjson string cert: Le certificat de l'entité

   **Exemple de requête**

       .. code-block:: http

          PUT https://my_waarp_gateway.net/api/certificates/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 104

          {
            "name": "certificat_sftp_new",
            "ownerType": "local_agents",
            "ownerID": 1,
            "privateKey": "<clé privée>",
            "publicKey": "<clé publique>",
            "cert": "<certificat>"
          }


   **Réponse**

   :statuscode 201: Le certificat a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat demandé n'existe pas

   :resheader Location: Le chemin d'accès au certificat modifié

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/certificates/1