Lister les clients
==================

.. http:get:: /api/clients/(string:client_name)

   Renvoie une liste des clients remplissant les critères donnés en paramètres
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

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array clients: La liste des clients demandés.
   :resjsonarr string name: Le nom du client.
   :resjsonarr string localAddress: L'adresse locale du client (en format [adresse:port])
   :resjsonarr object protoConfig: La configuration du client encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjsonarr array partners: La liste des partenaires rattachés au client. Voir
      :any:`rest_partners_list` pour plus de détails sur la structure de cette liste.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/clients?limit=10&sort=name- HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 619

         {
           "clients": [{
             "name": "sftp_client",
             "localAddress": "0.0.0.0:2222",
             "protoConfig": {},
             "partners": []
           },{
             "name": "r66_client",
             "localAddress": "0.0.0.0:6666",
             "protoConfig": {},
             "partners": []
           }]
         }
