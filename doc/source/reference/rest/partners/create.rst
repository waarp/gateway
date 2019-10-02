Créer un partenaire
===================

.. http:post:: /api/partners

   Ajoute un nouveau partenaire avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du partenaire
   :reqjson string protocol: Le protocole utilisé par le partenaire
   :reqjson string protoConfig: La configuration du partenaire encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/partners HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 88

          {
            "name": "waarp_sftp",
            "protocol": "sftp",
            "protoConfig": "{\"address\":\"waarp.org\",\"port\":21}
          }


   **Réponse**

   :statuscode 201: Le partenaire a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau partenaire créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/partners/1
