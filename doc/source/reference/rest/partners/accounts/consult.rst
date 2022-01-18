Consulter un compte distant
===========================

.. http:get:: /api/partners/(string:partner_name)/accounts/(string:login)

   Renvoie le compte ayant le login ``login`` rattaché au partner ``partner_name``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le compte a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resjson string login: Le login du compte
   :resjson array authMethods: La liste des valeurs utilisées par la gateway
      pour s'authentifier auprès du partenaire quand celle-ci s'y connecte.
   :resjson object authorizedRules: Les règles que le compte est autorisé à
      utiliser pour les transferts.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   **Exemple de réponse**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/partners/waarp_sftp/account/titi HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 152

         {
           "login": "titi",
           "authMethods": ["titi_private_key", "password"],
           "authorizedRules": {
             "sending": ["règle_envoi_1", "règle_envoi_2"],
             "reception": ["règle_récep_1", "règle_récep_2"]
           }
         }