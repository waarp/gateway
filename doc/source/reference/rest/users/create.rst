Ajouter un utilisateur
======================

.. http:post:: /api/users

   Ajoute un nouvel utilisateur avec les informations renseignées en format JSON
   dans le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string username: Le nom de l'utilisateur
   :reqjson string password: Le mot de passe de l'utilisateur
   :reqjson object perms: Les droits de l'utilisateur. Chaque attribut correspond
      à un élément sur lequel l' utilisateurs peut agir, et leur valeur indique
      les actions autorisées. Les différentes actions possibles sont lecture (*r*),
      écriture (*w*) et suppression (*d*). Ces droits sont renseignés avec une
      syntaxe similaire à `chmod <https://fr.wikipedia.org/wiki/Chmod#Modes>`_ où
      l'autorisation d'exécution a été remplacée par la suppression. Les opérateurs
      de changement d'état *+* et *-* peuvent également être utilisés pour
      ajouter ou retirer un type de droit aux droits courants, et l'opérateur *=*
      pour les écraser. (*Note*: l'opérateur *-* est inconséquent dans le cadre
      d'une création vu que l'utilisateur n'a pas de droits courants).

      * **transfers** (*string*) - Les droits sur les transferts. (*Note*:
         les transferts ne peuvent pas être supprimés).
      * **servers** (*string*) - Les droits sur les serveurs locaux.
      * **partners** (*string*) - Les droits sur les partenaires distants.
      * **rules** (*string*) - Les droits sur les règles de transfert.
      * **users** (*string*) - Les droits sur les autres utilisateurs.

   :statuscode 201: L'utilisateur a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres de l'utilisateur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouvel utilisateur créé

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/users HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 174

         {
           "username": "toto",
           "password": "titi",
           "perms": {
             "transfers":"=rw-",
             "servers":"=r--",
             "partners":"=r--",
             "rules":"=rwd",
             "users":"=---"
           }
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/users/toto