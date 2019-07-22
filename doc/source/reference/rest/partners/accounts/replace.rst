Remplacer un compte
===================

.. http:put:: /api/partners/(partner)/accounts/(account)

   Remplace le compte de l'utilisateur `account` rattaché au partenaire portant
   le nom `partner` avec les informations renseignées en format JSON dans le corps
   de la requête. Les champs non-spécifiés seront remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Request JSON Object:

       * **Username** (*string*) - Le nom d'utilisateur du compte
       * **Password** (*string*) - Le mot de passe du compte

   :Example:
       .. code-block:: http

          PUT /api/partners/partenaire1/accounts/utilisateur1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 68

          {
            "Username": "utilisateur1b",
            "Password": "nouveau_mot_de_passe"
          }


   **Réponse**

   :statuscode 201: Le compte a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte ou le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au compte modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1/accounts/utilisateur1b