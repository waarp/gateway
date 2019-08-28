Remplacer un compte
===================

.. http:put:: /api/accounts/(int:account_id)

   Remplace le compte portant le numéro ``account_id`` par celui renseigné
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront
   remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number ID: L'identifiant unique du compte
   :reqjson number PartnerID: L'identifiant unique du partenaire auquel le compte est rattaché
   :reqjson string Username: Le nom d'utilisateur du compte
   :reqjson string Password: Le mot de passe du compte

   **Exemple de requête**

       .. code-block:: http

          PUT /api/accounts/1234 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 96

          {
            "ID": 2345,
            "PartnerID": 23456,
            "Name": "partenaire1b",
            "Password": "nouveau_mot_de_passe"
          }


   **Réponse**

   :statuscode 201: Le compte a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resheader Location: Le chemin d'accès au compte modifié

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/accounts/2345