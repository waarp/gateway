Ajouter un compte
=================

.. http:post:: /api/accounts

   Ajoute un nouveau compte rattaché au partenaire numéro ``PartnerID``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number PartnerID: L'identifiant unique du partenaire auquel le compte est rattaché
   :reqjson string Username: Le nom d'utilisateur du compte
   :reqjson string Password: Le mot de passe du compte

   **Exemple de requête**

       .. code-block:: http

          POST /api/accounts HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 109

          {
            "PartnerID": 12345,
            "Username": "utilisateur1",
            "Password": "mot_de_passe"
          }


   **Réponse**

   :statuscode 201: Le compte a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau compte créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1/accounts/utilisateur1
