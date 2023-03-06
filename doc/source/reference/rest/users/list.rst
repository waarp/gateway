Lister les utilisateurs
=======================

.. http:get:: /api/users

   Renvoie une liste des utilisateurs remplissant les critères donnés en
   paramètres de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les utilisateurs seront triés *(défaut: username+)*
   :type sort: [username+|username-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array users: La liste des utilisateur demandés
   :resjsonarr string username: Le nom de l'utilisateur
   :resjsonarr object perms: Les droits de l'utilisateur. Chaque attribut correspond
      à un élément sur lequel l' utilisateurs peut agir, et leur valeur indique
      les actions autorisées. Les différentes actions possibles sont lecture (*r*),
      écriture (*w*) et suppression (*d*). Ces droits sont renseignés avec une
      syntaxe similaire à `chmod <https://fr.wikipedia.org/wiki/Chmod#Modes>`_ où
      l'autorisation d'exécution a été remplacée par la suppression.

      * **transfers** (*string*) - Les droits sur les transferts. (*Note*:
        les transferts ne peuvent pas être supprimés).
      * **servers** (*string*) - Les droits sur les serveurs locaux.
      * **partners** (*string*) - Les droits sur les partenaires distants.
      * **rules** (*string*) - Les droits sur les règles de transfert.
      * **users** (*string*) - Les droits sur les autres utilisateurs.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/users?limit=10&sort=username- HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 76

         {
           "users": [{
             "username": "tutu",
             "perms": {
               "transfers":"r--",
               "servers":"rw-",
               "partners":"rw-",
               "rules":"---",
               "users":"---"
             }
           },{
             "username": "toto",
             "perms": {
               "transfers":"rw-",
               "servers":"r--",
               "partners":"r--",
               "rules":"rwd",
               "users":"---"
             }
           }]
         }