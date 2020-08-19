Lister les partenaires
======================

.. http:get:: /api/partners

   Renvoie une liste des partenaires remplissant les critères donnés en paramètres
   de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les partenaires seront triés
      Valeurs possibles : ``name+``, ``name-``, ``protocol+``, ``protocol-``.
      *(défaut: name+)*
   :type sort: string
   :param protocol: Filtre uniquement les partenaires utilisant ce protocole.
      Peut être renseigné plusieurs fois pour filtrer plusieurs protocoles.
   :type protocol: string

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array partners: La liste des partenaires demandés
   :resjsonarr string name: Le nom du partenaire
   :resjsonarr string protocol: Le protocole utilisé par le partenaire
   :resjsonarr object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjsonarr object authorizedRules: Les règles que le partenaire est autorisé à
      utiliser pour les transferts.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/partners?limit=10&sort=name-&protocol=sftp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 619

         {
           "partners": [{
             "name": "waarp_sftp_2",
             "protocol": "sftp",
             "protoConfig": {
               "address": "waarp.org",
               "port": 22
             },
             "authorizedRules": {
               "sending": ["règle_envoi_1", "règle_envoi_2"],
               "reception": ["règle_récep_1", "règle_récep_2"]
             }
           },{
             "name": "waarp_sftp_1",
             "protocol": "sftp",
             "protoConfig": {
               "address": "waarp.org",
               "port": 21
             },
             "authorizedRules": {
               "sending": ["règle_envoi_1", "règle_envoi_2"],
               "reception": ["règle_récep_1", "règle_récep_2"]
             }
           }]
         }
