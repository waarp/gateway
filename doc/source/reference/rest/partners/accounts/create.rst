Ajouter un compte
=================

.. http:post:: /api/partners/(partner)/accounts

   Ajoute un nouveau compte rattaché au partenaire nommé `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Request JSON Object:

       * **Username** (*string*) - Le nom d'utilisateur du compte
       * **Password** (*string*) - Le mot de passe du compte

   :Example:
       .. code-block:: http

          POST /api/partners/partenaire1/accounts HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 88

          {
            "Username": "utilisateur1",
            "Password": "mot_de_passe"
          }


   **Réponse**

   :statuscode 201: Le compte a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au nouveau compte créé

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1/accounts/utilisateur1
