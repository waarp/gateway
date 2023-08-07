Ajouter un compte distant
=========================

.. http:post:: /api/partners/(string:partner_name)/accounts/(string:login)

   Ajoute un nouveau compte au partenaire donné avec les informations renseignées
   en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string login: Le login du compte
   :reqjson string password: Le mot de passe du compte

   :statuscode 201: Le compte a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau compte créé


   **Exemple de requête**

   .. code-block:: http

      POST https://my_waarp_gateway.net/api/partner/waarp_sftp/accounts HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 109

      {
        "login": "titi",
        "password": "titi"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/partners/waarp_sftp/accounts/titi
