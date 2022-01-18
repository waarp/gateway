Consulter un compte local
=========================

.. http:get:: /api/servers/(string:server_name)/accounts/(string:login)

   Renvoie le compte ayant le login ``login`` rattaché au server ``server_name``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le compte a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resjson string login: Le login du compte
   :resjson array authMethods: La liste des valeurs utilisées par le client
      pour s'authentifier auprès de la gateway quand il se connecte à celle-ci.
   :resjson object authorizedRules: Les règles que le compte est autorisé à
      utiliser pour les transferts.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/servers/sftp_server/account/toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 152

         {
           "login": "toto",
           "authMethods": ["password", "toto_public_key"],
           "authorizedRules": {
             "sending": ["règle_envoi_1", "règle_envoi_2"],
             "reception": ["règle_récep_1", "règle_récep_2"]
           }
         }