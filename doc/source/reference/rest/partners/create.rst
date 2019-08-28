Créer un partenaire
===================

.. http:post:: /api/partners

   Ajoute un nouveau partenaire avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string Name: Le nom du partenaire
   :reqjson string Address: L'address (IP ou DNS) du partenaire
   :reqjson number Port: Le port sur lequel le partenaire écoute
   :reqjson [sftp] Type: Le type de partenaire

   **Exemple de requête**

       .. code-block:: http

          GET /api/partners HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 88

          {
            "Name": "partenaire1",
            "Addresse": "waarp.fr",
            "Port": 21,
            "Type": "sftp"
          }


   **Réponse**

   :statuscode 201: Le partenaire a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau partenaire créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/1234
