Lister les comptes distants
===========================

.. http:get:: /api/partners/(string:partner_name)/accounts

   Renvoie une liste des comptes du partenaire donné en fonction des paramètres fournis.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les comptes seront triés *(défaut: login+)*
   :type sort: [login+|login-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array remoteAccounts: La liste des comptes demandés
   :resjsonarr string login: Le login du compte
   :resjsonarr array authMethods: La liste des valeurs utilisées par la gateway
      pour s'authentifier auprès du partenaire quand celle-ci s'y connecte.
   :resjsonarr object authorizedRules: Les règles que le compte est autorisé à
         utiliser pour les transferts.

         * **sending** (*array* of *string*) - Les règles d'envoi.
         * **reception** (*array* of *string*) - Les règles de réception.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/partners/waarp_sftp/accounts?limit=10&sort=name- HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 147

         {
           "remoteAccounts": [{
             "login": "tutu",
             "authMethods": ["tutu_private_key"],
             "authorizedRules": {
               "sending": ["règle_envoi_1", "règle_envoi_2"],
               "reception": ["règle_récep_1", "règle_récep_2"]
             }
           },{
             "login": "titi",
             "authMethods": ["titi_private_key", "password"],
             "authorizedRules": {
               "sending": ["règle_envoi_1", "règle_envoi_2"],
               "reception": ["règle_récep_1", "règle_récep_2"]
             }
           }]
         }